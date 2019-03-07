package generator

import (
    "encoding/base64"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "text/template"

    getter "github.com/hashicorp/go-getter"
    "gopkg.in/yaml.v2"
)

type generator struct {
    baseRepo       string
    basePath       string
    installerPath  string
    secretsRepo    string
    settingsPath   string
    buildPath      string
}

func New(baseRepo string, basePath string, installerPath string, secretsRepo string, settingsPath string, buildPath string) generator {
    g := generator {baseRepo, basePath, installerPath, secretsRepo, settingsPath, buildPath}
    return g
}

func (g generator) DownloadArtifacts() {
    // Download installer for openshift
    log.Println("Downloading openshift-install binary")
    binaryPath := fmt.Sprintf("%s/openshift-install", g.buildPath)
    client := &getter.Client{Src: g.installerPath, Dst: binaryPath}
    err := client.Get()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error downloading openshift-install binary: %s", err))
        os.Exit(1)
    }
    os.Chmod(binaryPath, 0777)

    // Download the credentials repo
    log.Println("Download secrets repo")
    secretsPath := fmt.Sprintf("%s/secrets", g.buildPath)

    // Retrieve private key and b64encode it
    rsaPrivateLocation := fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME"))
    priv, _ := ioutil.ReadFile(rsaPrivateLocation)
    sEnc := base64.StdEncoding.EncodeToString(priv)

    finalUrl := fmt.Sprintf("%s?sshkey=%s", g.secretsRepo, sEnc)
    client = &getter.Client{Src: finalUrl, Dst: secretsPath, Mode: getter.ClientModeAny}
    err = client.Get()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error downloading secrets repo: %s", err))
        os.Exit(1)
    }
    os.Chmod(secretsPath, 0700)

    // Download the settings.yaml and place it on build directory
    log.Println("Download settings file")
    settingsBuildPath := fmt.Sprintf("%s/settings.yaml", g.buildPath)
    client = &getter.Client{Src: g.settingsPath, Dst: settingsBuildPath, Mode: getter.ClientModeFile}
    err = client.Get()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error downloading settings.yaml: %s", err))
        os.Exit(1)
    }

    // Clone the base repository with manifests
    log.Println("Cloning the base repository with manifests")
    baseBuildPath := fmt.Sprintf("%s/manifests", g.buildPath)
    log.Println(g.basePath)
    client = &getter.Client{Src: g.baseRepo, Dst: baseBuildPath, Mode: getter.ClientModeAny}
    err = client.Get()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error cloning base repository with manifests: %s", err))
        os.Exit(1)
    }

}
func settings(s string) string {
    log.Println(s)
    return "abc"
}

func (g generator) GenerateFromInstall() {
    // First download the needed artifacts
    g.DownloadArtifacts()

    // Read install-config.yaml on the given path and parse it
    manifestsPath := fmt.Sprintf("%s/manifests/%s", g.buildPath, g.basePath)
    installPath := fmt.Sprintf("%s/install-config.yaml.go", manifestsPath)

    t, err := template.New("install-config.yaml.go").ParseFiles(installPath)
    if err != nil {
        log.Fatal(fmt.Sprintf("Error reading install file: %s", err))
        os.Exit(1)
    }

    // parse settings file
    yamlContent, err := ioutil.ReadFile(fmt.Sprintf("%s/settings.yaml", g.buildPath))
    if err != nil {
        log.Fatal(fmt.Sprintf("Error reading settings file: %s", err))
        os.Exit(1)
    }

    siteSettings := &map[string]map[string]interface{}{}
    err = yaml.Unmarshal(yamlContent, &siteSettings)
    if err != nil {
        log.Fatal(fmt.Sprintf("Error parsing settings yaml file: %s", err))
    }
    parsedSettings := (*siteSettings)["settings"]

    // Prepare the vars to be executed in the template
    var settings = map[string]string{
        "baseDomain": parsedSettings["baseDomain"].(string),
        "clusterName": parsedSettings["clusterName"].(string),
        "clusterCIDR": parsedSettings["clusterCIDR"].(string),
        "clusterSubnetLength": fmt.Sprintf("%s", parsedSettings["clusterSubnetLength"].(int)),
        "machineCIDR": parsedSettings["machineCIDR"].(string),
        "serviceCIDR": parsedSettings["serviceCIDR"].(string),
        "SDNType": parsedSettings["SDNType"].(string),
        "libvirtURI": parsedSettings["libvirtURI"].(string),
        "pullSecret": "pull",
        "SSHKey": "key",
    }
    err = t.Execute(os.	Stdout, settings)

    if err != nil {
        log.Fatal(fmt.Sprintf("Error parsing template: %s", err))
    }

}
