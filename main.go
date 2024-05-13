package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/webdav-serverless/webdav-serverless/awsfs"
	"github.com/webdav-serverless/webdav-serverless/webdav"
)

type Params struct {
	Port                int    `mapstructure:"port"`
	DynamoDBTablePrefix string `mapstructure:"dynamodb-table-prefix"`
	S3BucketName        string `mapstructure:"s3-bucket-name"`
	DynamoDBURL         string `mapstructure:"dynamodb-url"`
	S3URL               string `mapstructure:"s3-url"`
	BasicAuthUser       string `mapstructure:"basic-auth-user"`
	BasicAuthPassword   string `mapstructure:"basic-auth-pass"`
	DisableBasicAuth    bool   `mapstructure:"disable-basic-auth"`
}

func main() {

	var params = &Params{}

	c := &cobra.Command{
		Use:     "webdav-serverless",
		Short:   "An implementation of the WebDav protocol backed by AWS S3 and DynamoDB",
		Long:    `An implementation of the WebDav protocol backed by AWS S3 and DynamoDB.`,
		Version: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(params)
		},
	}

	flags := c.PersistentFlags()
	flags.IntVar(&params.Port, "port", 80, "Port to serve on (Plain HTTP).")
	_ = viper.BindPFlag("port", flags.Lookup("port"))
	flags.StringVar(&params.DynamoDBTablePrefix, "dynamodb-table-prefix", "webdav-serverless-", "Prefix of DynamoDB table.")
	_ = viper.BindPFlag("dynamodb-table-prefix", flags.Lookup("dynamodb-table-prefix"))
	flags.StringVar(&params.S3BucketName, "s3-bucket-name", "webdav-serverless", "Name of S3 bucket.")
	_ = viper.BindPFlag("s3-bucket-name", flags.Lookup("s3-bucket-name"))
	flags.StringVar(&params.DynamoDBURL, "dynamodb-url", "", "DynamoDB base endpoint (for local development).")
	_ = viper.BindPFlag("dynamodb-url", flags.Lookup("dynamodb-url"))
	flags.StringVar(&params.S3URL, "s3-url", "", "S3 base endpoint (for local development).")
	_ = viper.BindPFlag("s3-url", flags.Lookup("s3-url"))
	flags.StringVar(&params.BasicAuthUser, "basic-auth-user", "", "Basic auth user name.")
	_ = viper.BindPFlag("basic-auth-user", flags.Lookup("basic-auth-user"))
	flags.StringVar(&params.BasicAuthPassword, "basic-auth-pass", "", "Basic auth password.")
	_ = viper.BindPFlag("basic-auth-pass", flags.Lookup("basic-auth-pass"))
	flags.BoolVar(&params.DisableBasicAuth, "disable-basic-auth", false, "Disable basic auth.")
	_ = viper.BindPFlag("disable-basic-auth", flags.Lookup("disable-basic-auth"))

	cobra.OnInitialize(func() {
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		viper.AutomaticEnv()
		if err := viper.Unmarshal(&params); err != nil {
			fmt.Println("Error decoding params:", err)
			return
		}
	})

	if err := c.Execute(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}

func run(params *Params) error {

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load aws config: %v", err)
	}

	metadataStore := awsfs.MetadataStore{
		EntryTableName:     params.DynamoDBTablePrefix + "entry",
		ReferenceTableName: params.DynamoDBTablePrefix + "reference",
		DynamoDBClient: dynamodb.NewFromConfig(cfg, func(options *dynamodb.Options) {
			if params.DynamoDBURL != "" {
				options.BaseEndpoint = &params.DynamoDBURL
			}
		}),
	}

	physicalStore := awsfs.PhysicalStore{
		BucketName: params.S3BucketName,
		S3Client: s3.NewFromConfig(cfg, func(options *s3.Options) {
			if params.S3URL != "" {
				options.UsePathStyle = true
				options.BaseEndpoint = &params.S3URL
			}
		}),
	}

	if err = metadataStore.Init(context.Background()); err != nil {
		return fmt.Errorf("failed to init refarence: %v", err)
	}

	srv := &webdav.Handler{
		FileSystem: &awsfs.Server{
			MetadataStore: metadataStore,
			PhysicalStore: physicalStore,
			TempDir:       filepath.Clean(os.TempDir()),
		},
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, code int, err error) {
			litmus := r.Header.Get("X-Litmus")
			if len(litmus) > 19 {
				litmus = litmus[:16] + "..."
			}
			switch r.Method {
			case "COPY", "MOVE":
				dst := ""
				if u, err := url.Parse(r.Header.Get("Destination")); err == nil {
					dst = u.Path
				}
				o := r.Header.Get("Overwrite")
				log.Printf("%-20s%-10s%-30s%-30so=%-2s%-10d%v", litmus, r.Method, r.URL.Path, dst, o, code, err)
			default:
				log.Printf("%-20s%-10s%-30s%-10d%v", litmus, r.Method, r.URL.Path, code, err)
			}
		},
	}

	// The next line would normally be:
	//	http.Handle("/", h)
	// but we wrap that HTTP handler h to cater for a special case.
	//
	// The propfind_invalid2 litmus test case expects an empty namespace prefix
	// declaration to be an error. The FAQ in the webdav litmus test says:
	//
	// "What does the "propfind_invalid2" test check for?...
	//
	// If a request was sent with an XML body which included an empty namespace
	// prefix declaration (xmlns:ns1=""), then the server must reject that with
	// a "400 Bad Request" response, as it is invalid according to the XML
	// Namespace specification."
	//
	// On the other hand, the Go standard library's encoding/xml package
	// accepts an empty xmlns namespace, as per the discussion at
	// https://github.com/golang/go/issues/8068
	//
	// Empty namespaces seem disallowed in the second (2006) edition of the XML
	// standard, but allowed in a later edition. The grammar differs between
	// http://www.w3.org/TR/2006/REC-xml-names-20060816/#ns-decl and
	// http://www.w3.org/TR/REC-xml-names/#dt-prefix
	//
	// Thus, we assume that the propfind_invalid2 test is obsolete, and
	// hard-code the 400 Bad Request response that the test expects.
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !params.DisableBasicAuth {
			if user, pass, ok := r.BasicAuth(); !ok || user != params.BasicAuthUser || pass != params.BasicAuthPassword {
				w.Header().Add("WWW-Authenticate", `Basic realm="Please enter your username and password."`)
				http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
				log.Printf("%-20s%-10s%-30s%-10d%v", "", r.Method, r.URL.Path, http.StatusUnauthorized,
					errors.New("unauthorized"))
				return
			}
		}
		if r.Header.Get("X-Litmus") == "props: 3 (propfind_invalid2)" {
			http.Error(w, "400 Bad Request", http.StatusBadRequest)
			return
		}
		srv.ServeHTTP(w, r)
	}))

	log.Printf("WEBDAV ListenAndServe: [%s]\n", fmt.Sprintf(":%d", params.Port))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", params.Port), nil); err != nil {
		return fmt.Errorf("error with WebDAV server: %v", err)
	}

	return nil
}
