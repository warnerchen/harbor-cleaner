package main

import (
    "crypto/tls"
    "encoding/json"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
    "io"
)

// Repository defines the structure of a repository object
type Repository struct {
    Name string `json:"name"`
}

// Artifact defines the structure of an artifact object
type Artifact struct {
    Digest string `json:"digest"`
    Tags   []struct {
        Name string `json:"name"`
    } `json:"tags"`
}

// Check if the project exists in the registry
func checkProjectExists(registryURL, registryUsername, registryPassword, project string) bool {
    getProjectURL := registryURL + "/api/v2.0/projects/" + project
    transport := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{
        Transport: transport,
        Timeout:   10 * time.Second,
    }

    req, err := http.NewRequest("GET", getProjectURL, nil)
    if err != nil {
        log.Fatalf("Error creating request: %v", err)
    }

    req.Header.Add("accept", "application/json")
    req.Header.Add("X-Is-Resource-Name", "false")
    req.SetBasicAuth(registryUsername, registryPassword)

    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("Error sending request: %v", err)
    }
    defer resp.Body.Close()

    return resp.StatusCode == http.StatusOK
}

// Get all repositories for a given project
func getAllRepositories(registryURL, registryUsername, registryPassword, project string) []string {
    getRepositoryURL := registryURL + "/api/v2.0/projects/" + project + "/repositories?page=-1&page_size=-1"
    transport := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{
        Transport: transport,
        Timeout:   10 * time.Second,
    }

    req, err := http.NewRequest("GET", getRepositoryURL, nil)
    if err != nil {
        log.Fatalf("Error creating request: %v", err)
    }

    req.Header.Add("accept", "application/json")
    req.SetBasicAuth(registryUsername, registryPassword)

    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("Error sending request: %v", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Error reading response body: %v", err)
    }

    var repositories []Repository
    err = json.Unmarshal(body, &repositories)
    if err != nil {
        log.Fatalf("Error parsing JSON response: %v", err)
    }

    var repositoryList []string
    for _, repo := range repositories {
        parts := strings.Split(repo.Name, "/")
        repositoryList = append(repositoryList, parts[len(parts)-1])
    }

    return repositoryList
}

// Delete an artifact from the repository
func cleanTag(registryURL, registryUsername, registryPassword, project, repository string, tagList []string) {
    transport := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{
        Transport: transport,
        Timeout:   10 * time.Second,
    }

    getArtifactUrl := registryURL + "/api/v2.0/projects/" + project + "/repositories/" + repository + "/artifacts?page=-1&page_size=-1&with_tag=true"
    
    req, err := http.NewRequest("GET", getArtifactUrl, nil)
    if err != nil {
        log.Fatalf("Error creating request: %v", err)
    }

    req.Header.Add("accept", "application/json")
    req.SetBasicAuth(registryUsername, registryPassword)

    resp, err := client.Do(req)
    if err != nil {
        log.Fatalf("Error sending request: %v", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Error reading response body: %v", err)
    }

    var artifacts []Artifact
    if err := json.Unmarshal(body, &artifacts); err != nil {
        log.Fatalf("Error parsing JSON response: %v", err)
    }

    excludeTags := []string{"latest", "arch"}

    digestMap := make(map[string][]string)

    for _, artifact := range artifacts {
        allTagsExcluded := true

        for _, tag := range artifact.Tags {
            excluded := false

            for _, excludeTag := range excludeTags {
                if strings.Contains(tag.Name, excludeTag) {
                    log.Printf("Skipping tag %s (Digest: %s) as it matches exclusion pattern %s...",
                        tag.Name, artifact.Digest, excludeTag)
                    excluded = true
                    break
                }
            }

            if excluded {
                continue
            }

            allTagsExcluded = false

            for _, keyword := range tagList {
                if strings.Contains(tag.Name, keyword) {
                    digestMap[artifact.Digest] = append(digestMap[artifact.Digest], tag.Name)
                    log.Printf("Match found: Tag %s (Digest: %s) contains keyword %s, adding to delete list...",
                        tag.Name, artifact.Digest, keyword)
                }
            }
        }

        if allTagsExcluded {
            log.Printf("Skipping entire artifact (Digest: %s) as all its tags are excluded.", artifact.Digest)
        }
    }

    if len(digestMap) > 0 {
        log.Printf("Artifacts to delete in repository %s:", repository)
        for digest, tags := range digestMap {
            log.Printf("Digest: %s, Tags: %v", digest, tags)
        }
    }

    for digest, tags := range digestMap {
        for _, tag := range tags {
            deleteTagURL := registryURL + "/api/v2.0/projects/" + project + "/repositories/" + repository + "/artifacts/" + digest + "/tags/" + tag

            req, err = http.NewRequest("DELETE", deleteTagURL, nil)
            if err != nil {
                log.Printf("Error creating DELETE request for %s: %v", deleteTagURL, err)
                continue
            }

            req.Header.Add("accept", "application/json")
            req.SetBasicAuth(registryUsername, registryPassword)

            resp, err = client.Do(req)
            if err != nil {
                log.Printf("Error sending DELETE request for %s: %v", deleteTagURL, err)
                continue
            }
            defer resp.Body.Close()

            if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
                log.Printf("Successfully deleted tag %s (Digest: %s)", tag, digest)
            } else {
                log.Printf("Failed to delete tag %s (Digest: %s), status code: %d", tag, digest, resp.StatusCode)
            }
        }
    }
}

func main() {
    registryURL := os.Getenv("HARBOR_REGISTRY")
    registryUsername := os.Getenv("HARBOR_USERNAME")
    registryPassword := os.Getenv("HARBOR_PASSWORD")
    projects := os.Getenv("HARBOR_PROJECTS")
    tags := os.Getenv("HARBOR_TAGS")

    if registryURL == "" || registryUsername == "" || registryPassword == "" || projects == "" || tags == "" {
        log.Fatal("HARBOR_REGISTRY, HARBOR_USERNAME, HARBOR_PASSWORD, HARBOR_PROJECTS, and HARBOR_TAGS environment variables must be set")
    }

    projectList := strings.Split(projects, ",")
    tagList := strings.Split(tags, ",")

    for _, project := range projectList {
        if checkProjectExists(registryURL, registryUsername, registryPassword, project) {
            log.Printf("Project '%s' exists. Begin cleaning specific tag...", project)

            repositoryList := getAllRepositories(registryURL, registryUsername, registryPassword, project)
			for _, repository := range repositoryList {
				log.Printf("Cleaning repository '%s'...", repository)
                cleanTag(registryURL, registryUsername, registryPassword, project, repository, tagList)
			}
        } else {
            log.Printf("Project '%s' does not exist or failed to retrieve. Skipping...", project)
        }
    }
}
