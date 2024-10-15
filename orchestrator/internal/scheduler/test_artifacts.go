package scheduler

import (
	"log"

	"my.org/novel_vmp/data"
)

func getTestArtifact(inputArtifactType string) *data.Artifact {
	var artifact *data.Artifact
	switch inputArtifactType {
	case data.ArtifactTypeHost:
		artifact = &data.Artifact{
			ArtifactType: data.ArtifactTypeHost,
			Value:        "localhost:8081",
			Scanner:      "config",
		}
	case data.ArtifactTypeIP:
		artifact = &data.Artifact{
			ArtifactType: data.ArtifactTypeIP,
			Value:        "127.0.0.1",
			Scanner:      "config",
		}
	case data.ArtifactTypeDomain:
		artifact = &data.Artifact{
			ArtifactType: data.ArtifactTypeDomain,
			Value:        "localhost",
			Scanner:      "config",
		}
	case data.ArtifactTypeURL:
		artifact = &data.Artifact{
			ArtifactType: data.ArtifactTypeURL,
			Value:        "http://localhost:8081/",
			Scanner:      "config",
		}
	case data.ArtifactTypeHttpMsg:
		artifact = &data.Artifact{
			ArtifactType: data.ArtifactTypeHttpMsg,
			Scanner:      "config",
			Value:        "http://localhost:3000/",
			Location: data.Location{
				URL: "http://localhost:3000/",
			},
			Response: `HTTP/1.1 200 OK
Age: 307112
Cache-Control: max-age=604800
Content-Type: text/html; charset=UTF-8
Date: Mon, 14 Oct 2024 08:02:16 GMT
Etag: "3147526947+gzip+ident"
Expires: Mon, 21 Oct 2024 08:02:16 GMT
Last-Modified: Thu, 17 Oct 2019 07:18:26 GMT
Server: ECAcc (dcd/7D26)
Vary: Accept-Encoding
X-Cache: HIT
Content-Length: 1256

<!doctype html>
<html>
<head>
    <title>Example Domain</title>

    <meta charset="utf-8" />
    <meta http-equiv="Content-type" content="text/html; charset=utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style type="text/css">
    body {
        background-color: #f0f0f2;
        margin: 0;
        padding: 0;
        font-family: -apple-system, system-ui, BlinkMacSystemFont, "Segoe UI", "Open Sans", "Helvetica Neue", Helvetica, Arial, sans-serif;
        
    }
    div {
        width: 600px;
        margin: 5em auto;
        padding: 2em;
        background-color: #fdfdff;
        border-radius: 0.5em;
        box-shadow: 2px 3px 7px 2px rgba(0,0,0,0.02);
    }
    a:link, a:visited {
        color: #38488f;
        text-decoration: none;
    }
    @media (max-width: 700px) {
        div {
            margin: 0 auto;
            width: auto;
        }
    }
    </style>    
</head>

<body>
<div>
    <h1>Example Domain</h1>
    <p>This domain is for use in illustrative examples in documents. You may use this
    domain in literature without prior coordination or asking for permission.</p>
    <p><a href="https://www.iana.org/domains/example">More information...</a></p>
</div>
</body>
</html>`,
		}
	default:
		log.Fatalf("Test artifact not configured for: %v", inputArtifactType)
	}
	return artifact
}
