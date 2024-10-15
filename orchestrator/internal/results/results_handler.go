package results

import (
	"context"
	"log"
	"strings"
	"time"

	"my.org/novel_vmp/data"

	"github.com/elastic/go-elasticsearch/v8"
)

func UploadArtifactsToElastic(artifacts []*data.Artifact) {
	typedClient, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
		Username:  "elastic",
		Password:  "changeme",
	})
	if err != nil {
		log.Fatal(err)
	}
	datetime := strings.ReplaceAll(strings.ReplaceAll(time.Now().Format(time.DateTime), " ", "_"), ":", "-")
	index_name := "NOVELVMP_" + datetime
	_, err = typedClient.Indices.Create(index_name).Do(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("ResultsHandler: sending results to elasticsearch: count = ", len(artifacts))

	for _, artifact := range artifacts {
		_, err := typedClient.Index(index_name).
			Request(artifact).
			Do(context.TODO())
		if err != nil {
			log.Print(err)
		}
	}
	log.Println("ResultsHandler: finished sending results to elasticsearch")

}
