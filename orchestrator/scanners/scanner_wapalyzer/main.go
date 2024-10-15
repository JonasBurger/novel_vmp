package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
	"my.org/novel_vmp/data"
	"my.org/novel_vmp/pkg/slave"
)

const scannerName string = "scanner_wapalyzer"

var wappalyzerClient *wappalyzer.Wappalyze

func main() {

	var err error
	wappalyzerClient, err = wappalyzer.New()
	if err != nil {
		log.Panic("panic: ", err)
	}

	err = slave.NewServer(work).Start()
	if err != nil {
		log.Panic("panic: ", err)
	}
}

func work(artifact *data.Artifact) {
	if artifact.ArtifactType != data.ArtifactTypeHttpMsg {
		log.Panic("panic: ", scannerName, ": artifact type not supported: ", artifact.ArtifactType)
	}

	buf := bytes.NewBuffer([]byte(artifact.Response))
	reader := bufio.NewReader(buf)

	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		log.Println("Error reading response:", err)
		log.Println("Thus skipping this artifact: ", artifact.Location.URL)
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic("Error reading body:", err)
	}

	fingerprints := wappalyzerClient.Fingerprint(resp.Header, bodyBytes)
	fmt.Printf("%v\n", fingerprints)
	fingerprints_with_cats := wappalyzerClient.FingerprintWithCats(resp.Header, bodyBytes)

	for fingerprint := range fingerprints {

		app, version := separateAppVersion(fingerprint)
		artifact := data.Artifact{
			ArtifactType: data.ArtifactTypeTechnology,
			Scanner:      scannerName,
			Value:        app,
			Location:     artifact.Location,
		}
		if cats, ok := fingerprints_with_cats[fingerprint]; ok {
			cats, groups := getCatsAndGroups(cats.Cats)
			artifact.AdditionalData = map[string]interface{}{
				"categories": cats,
				"groups":     groups,
			}
		}

		if version != "" {
			artifact.Version = version
		}
		slave.SendArtifact(artifact)
	}

}

func separateAppVersion(value string) (string, string) {
	if strings.Contains(value, ":") {
		if parts := strings.Split(value, ":"); len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return value, ""
}

type category struct {
	Name     string
	Priority int
	Groups   []string
}

type categoryJson struct {
	Groups   []int  `json:"groups"`
	Name     string `json:"name"`
	Priority int    `json:"priority"`
}

type groupJson struct {
	Name string `json:"name"`
}

var categories = loadCategories()

func getCatsAndGroups(catsIn []int) ([]string, []string) {
	cats := []string{}
	groups := []string{}
	for _, catId := range catsIn {
		cat, ok := categories[catId]
		if !ok {
			log.Panicf("Category not found: %v", catId)
		}
		cats = append(cats, cat.Name)
		groups = append(groups, cat.Groups...)
	}
	return removeDuplicates(cats), removeDuplicates(groups)
}

func removeDuplicates[T comparable](s []T) []T {
	seen := make(map[T]struct{})
	var result []T

	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func loadCategories() map[int]category {
	file, err := os.Open("wapalyzer_data/categories.json")
	if err != nil {
		log.Panicf("Error opening file: %v", err)
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		log.Panicf("Error reading file: %v", err)
	}

	categories := make(map[string]categoryJson)
	err = json.Unmarshal(byteValue, &categories)
	if err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	file, err = os.Open("wapalyzer_data/groups.json")
	if err != nil {
		log.Panicf("Error opening file: %v", err)
	}
	defer file.Close()

	byteValue, err = io.ReadAll(file)
	if err != nil {
		log.Panicf("Error reading file: %v", err)
	}

	groups := make(map[string]groupJson)
	err = json.Unmarshal(byteValue, &groups)
	if err != nil {
		log.Panicf("Error parsing JSON: %v", err)
	}

	cats := make(map[int]category)
	for id, catJson := range categories {
		cat := category{
			Name:     catJson.Name,
			Priority: catJson.Priority,
			Groups:   []string{},
		}
		for gid, grp := range groups {
			if contains(cat.Groups, gid) {
				cat.Groups = append(cat.Groups, grp.Name)
			}
		}
		iid, err := strconv.Atoi(id)
		if err != nil {
			log.Panicf("Error parsing id: %v", err)
		}
		cats[iid] = cat
	}
	return cats
}

func contains(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
