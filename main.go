package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/webdav-serverless/webdav-serverless/awsfs"
	"github.com/webdav-serverless/webdav-serverless/webdav"
)

func main() {

	httpPort := flag.Int("port", 80, "Port to serve on (Plain HTTP)")
	httpsPort := flag.Int("port-secure", 443, "Port to serve TLS on")
	serveSecure := flag.Bool("secure", false, "Serve HTTPS. Default false")
	dynamodbURL := flag.String("dynamodb-url", "", "DynamoDB base endpoint (for local development)")
	s3URL := flag.String("s3-url", "", "S3 base endpoint (for local development)")

	flag.Parse()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load aws config: %v", err)
	}

	metadataStore := awsfs.MetadataStore{
		EntryTableName:     "Entry",
		ReferenceTableName: "Reference",
		DynamoDBClient: dynamodb.NewFromConfig(cfg, func(options *dynamodb.Options) {
			if *dynamodbURL != "" {
				options.BaseEndpoint = dynamodbURL
			}
		}),
	}

	s3Cfg := aws.Config{
		Region:      cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider("root", "deadbeef", ""),
	}

	physicalStore := awsfs.PhysicalStore{
		BucketName: "test",
		S3Client: s3.NewFromConfig(s3Cfg, func(options *s3.Options) {
			if *s3URL != "" {
				options.UsePathStyle = true
				options.BaseEndpoint = s3URL
			}
		}),
	}

	if err = metadataStore.Init(context.Background()); err != nil {
		log.Fatalf("failed to init refarence: %v", err)
	}

	srv := &webdav.Handler{
		FileSystem: awsfs.Server{
			MetadataStore: metadataStore,
			PhysicalStore: physicalStore,
			TempDir:       filepath.Clean(os.TempDir()),
		},
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, code int, err error) {
			litmus := r.Header.Get("X-Litmus")
			//if len(litmus) > 19 {
			//	litmus = litmus[:16] + "..."
			//}
			switch r.Method {
			case "COPY", "MOVE":
				dst := ""
				if u, err := url.Parse(r.Header.Get("Destination")); err == nil {
					dst = u.Path
				}
				o := r.Header.Get("Overwrite")
				log.Printf("%-20s%-10s%-30s%-30so=%-2s%v", litmus, r.Method, r.URL.Path, dst, o, err)
			default:
				log.Printf("%-20s%-10s%-30s%v", litmus, r.Method, r.URL.Path, err)
			}
			//if err != nil {
			//	log.Printf("WEBDAV [%s]: %s, %d, ERROR: %s\n", r.Method, r.URL, code, err)
			//} else {
			//	log.Printf("WEBDAV [%s]: %s, %d \n", r.Method, r.URL, code)
			//}
		},
	}
	http.Handle("/", srv)
	if *serveSecure == true {
		if _, err := os.Stat("./cert.pem"); err != nil {
			fmt.Println("[x] No cert.pem in current directory. Please provide a valid cert")
			return
		}
		if _, er := os.Stat("./key.pem"); er != nil {
			fmt.Println("[x] No key.pem in current directory. Please provide a valid cert")
			return
		}

		go func() {
			_ = http.ListenAndServeTLS(fmt.Sprintf(":%d", *httpsPort), "cert.pem", "key.pem", nil)
		}()
	}
	log.Printf("WEBDAV ListenAndServe: [%s]\n", fmt.Sprintf(":%d", *httpPort))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), nil); err != nil {
		log.Fatalf("Error with WebDAV server: %v", err)
	}

}
