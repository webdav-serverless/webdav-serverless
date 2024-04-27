# webdav-serverless

<p align="center">
  <img height="300" src="https://cacoo.com/diagrams/UZhoJO49E6jo81tL-C9964.png">
</p>

An implementation of the WebDav protocol backed by AWS S3 and DynamoDB.

## About the Design of webdav-serverless

### Diagram of webdav-serverless

<p align="center">
  <img src="https://cacoo.com/diagrams/UZhoJO49E6jo81tL-4254D.png">
</p>

### Metadata Structure Overview

**Metadata：**

![Metadata](https://cacoo.com/diagrams/UZhoJO49E6jo81tL-0511B.png)

**Reference：**

![Reference](https://cacoo.com/diagrams/UZhoJO49E6jo81tL-83487.png)

Note: In reality, the reference paths are hashed.

### Defining MetadataStore tables using DynamoDB

**Metadata：**

| Key    | Attributes         | Type   | Description                                     |
|--------|--------------------|--------|-------------------------------------------------|
| PK     | id                 | string | Unique ID (eg. UUID)                            |
| GSIPK1 | parent_id          | string | ID of the parent directory                      |
|        | name               | string | Name (eg. report.pdf)                           |
|        | type               | string | File system entry type (eg. File or Directory)  |
|        | size               | number | File size (eg. 512)                             |
|        | modify             | string | File modify time (eg. ISO 8601)                 |
|        | version            | number | Version number for optimistic lock (eg. 1)      |

**Reference：**

| Key    | Attributes         | Type   | Description                                |
|--------|--------------------|--------|--------------------------------------------|
| PK     | id                 | string | Unique ID (eg. hashed path)                |
|        | entries            | map    | key(hashed path): value(metadata id)       |
|        | version            | number | Version number for optimistic lock (eg. 1) |

### PhysicalStorage specifications using S3

```
# S3 Key (Metadata#id)
$bucket_name/$UUID
```

## Development

Starting Docker Compose:
```bash
docker-compose up --force-recreate --build --abort-on-container-exit
```

Initialize DynamoDB(dynamodb-local):
```bash
docker-compose run dynamodb-init
```

Initialize S3(minio):
```bash
docker-compose run s3-init
```

Run Go Application:
```bash
env AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE" AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" AWS_REGION="us-east-1" go run main.go --port=8080 --dynamodb-url=http://localhost:18070 --s3-url=http://localhost:19010
```

## Authors

* **[vvatanabe](https://github.com/vvatanabe/)** - *Main contributor*
* **[safx](https://github.com/safx/)** - *Main contributor*
* **[kunst1080](https://github.com/kunst1080/)** - *Main contributor*
* Currently, there are no other contributors

## License

This project is licensed under the MIT License. For detailed licensing information, refer to the [LICENSE](LICENSE) file included in the repository.
