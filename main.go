package main

import (
	"flag"
	"fmt"
	"github.com/nyudlts/go-aspace"
	"log"
	"net/url"
	"os"
	"strings"
)

var (
	domains []string
	repository int
	environment string
	config string
	client *aspace.ASClient
	err error
	update bool
	test bool
)

func init() {
	flag.IntVar(&repository,  "repository", 0, "repository")
	flag.StringVar(&environment, "environment", "", "environment")
	flag.StringVar(&config, "config", "/etc/go-aspace.yml", "config")
	flag.BoolVar(&test, "test", false, "test")
}

type DoInfo struct {
	DoID 	int
	RepoID 	int
}

type FileVersionInfo struct {
	URI string
	FVURI string
	FVUse string
}

func main() {
	flag.Parse()
	domains = []string{"webarchives.cdlib.org", "wayback.archive-it.org", "archive-it.org"}
	logfilename := fmt.Sprintf("webarchives-update-%s-repository-%d.log", environment, repository)
	fmt.Println("Running, logging to", logfilename)
	f, err := os.OpenFile(logfilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error creating logfile: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Println("INFO", "webarchive update tool")

	client, err = aspace.NewClient(config, environment, 20)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("INFO using go-aspace", aspace.LibraryVersion)


	doInfos := []DoInfo{}

	for _, i := range []int{2,3,6} {
		doids, err := client.GetDigitalObjectIDs(i)
		if err != nil {
			log.Fatal(err)
		}

		for _,doid := range doids {
			doInfos = append(doInfos, DoInfo{doid, i})
		}
	}

	doChunks := chunkDos(doInfos)

	fvChannel := make(chan []FileVersionInfo)

	for i := range doChunks {
		go getDOs(doChunks[i], fvChannel, i+1)
	}

	var results []FileVersionInfo

	for range doChunks {
		chunk := <- fvChannel
		log.Println("INFO Adding", len(chunk), "fvs to fv list")
		results = append(results, chunk...)
	}

	for i := range results {
		fmt.Println(results[i])
	}
}

func getDOs(chunk []DoInfo, fvChannel chan []FileVersionInfo, wID int) {
	fvinfos := []FileVersionInfo{}
	fmt.Println("starting worker", wID)
	for _,do := range chunk {
		d, err := client.GetDigitalObject(do.RepoID, do.DoID)
		if err != nil {
			log.Println("ERROR", err.Error())
			continue
		}

		if len(d.FileVersions) > 0 {
			for i := range d.FileVersions {
				fv := d.FileVersions[i]
				u, err := url.Parse(strings.TrimSpace(fv.FileURI))
				if err != nil {
					fmt.Println("Error", d.URI, err.Error())
					continue
				}
				fvinfos = append(fvinfos, FileVersionInfo{
					URI:  	d.URI,
					FVURI: 	u.Host,
					FVUse: 	fv.UseStatement,
				})
			}
		}

	}
	fvChannel <- fvinfos
}


func chunkDos(doinfos []DoInfo) [][]DoInfo {
	var divided [][]DoInfo

	chunkSize := (len(doinfos) + 7) / 8

	for i := 0; i < len(doinfos); i += chunkSize {
		end := i + chunkSize

		if end > len(doinfos) {
			end = len(doinfos)
		}

		divided = append(divided, doinfos[i:end])
	}
	return divided
}

	/*
	for _, doId := range doIds {

		do, err := client.GetDigitalObject(repository, doId)
		if err != nil {
			log.Printf("ERROR could not retrieve do %d", doId)
			break
		}

		log.Printf("INFO checking %s", do.URI)
		update = false

		fileversions := do.FileVersions

		for i,fv := range do.FileVersions {
			uri := strings.TrimSpace(fv.FileURI)
			uri = strings.Replace(uri, "\n", "", -1)
			u, err := url.Parse(uri)
			if err != nil {
				log.Printf("ERROR could not parse uri %s, skipping", fv.FileURI)
				break
			}

			if contains(u.Host) && fv.UseStatement == "service" {
				update = true
				fileversions[i].UseStatement = "external-link"
			}
		}

		if update == true {
			log.Printf("INFO Updating %s", do.URI)
			do.FileVersions = fileversions
			if test == false {
				msg, err := client.UpdateDigitalObject(repository, doId, do)
				if err != nil {
					log.Printf("ERROR %s", err)

				}
				log.Printf("INFO %s", msg)
			}
		} else {
			log.Printf("INFO %s conforms to existing rules", do.URI)
		}
	}
}

func contains(s string) bool {
	for _, domain := range domains {
		if s == domain {
			return true
		}
	}
	return false
}

	 */
