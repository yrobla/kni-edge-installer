package generator

import (
    "encoding/base64"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
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
    secrets        map[string]string

}

func New(baseRepo string, basePath string, installerPath string, secretsRepo string, settingsPath string, buildPath string) generator {
    g := generator {baseRepo, basePath, installerPath, secretsRepo, settingsPath, buildPath, make(map[string]string)}
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

    // Clone the base repository with base manifests
    log.Println("Cloning the base repository with base manifests")
    baseBuildPath := fmt.Sprintf("%s/base_manifests", g.buildPath)
    log.Println(g.basePath)
    client = &getter.Client{Src: g.baseRepo, Dst: baseBuildPath, Mode: getter.ClientModeAny}
    err = client.Get()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error cloning base repository with base manifests: %s", err))
        os.Exit(1)
    }

}

// traverse secrets directory and read content
func (g generator) ReadSecretFiles(path string, info os.FileInfo, err error) error {
    var matches = map[string]string{ "coreos-pull-secret": "pullSecret", "ssh-pub-key": "SSHKey"}

    if info.IsDir() && info.Name()== ".git" {
        return filepath.SkipDir
    }

    if err != nil {
        log.Fatal(fmt.Sprintf("Error traversing file: %s", err))
        os.Exit(1)
    }

    if ! info.IsDir() {
        data, err := ioutil.ReadFile(path)
        if err != nil {
            log.Fatal(fmt.Sprintf("Error reading file content: %s", err))
            os.Exit(1)
        }
        g.secrets[matches[info.Name()]] = strings.Trim(string(data), "\n")
    }
    return nil
}

func (g generator) GenerateInstallConfig() {
    // Read install-config.yaml on the given path and parse it
    manifestsPath := fmt.Sprintf("%s/base_manifests/%s", g.buildPath, g.basePath)
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
        os.Exit(1)
    }
    parsedSettings := (*siteSettings)["settings"]

    // Read secrets
    err = filepath.Walk(fmt.Sprintf("%s/secrets", g.buildPath), g.ReadSecretFiles)

    // Prepare the final file to write the template
    f, err := os.Create(fmt.Sprintf("%s/install-config.yaml", g.buildPath))
    if err != nil {
        log.Fatal(fmt.Sprintf("Error opening the install file: %s", err))
        os.Exit(1)
    }

    // Prepare the vars to be executed in the template
    var settings = map[string]string{
        "baseDomain": parsedSettings["baseDomain"].(string),
        "clusterName": parsedSettings["clusterName"].(string),
        "clusterCIDR": parsedSettings["clusterCIDR"].(string),
        "clusterSubnetLength": fmt.Sprintf("%d", parsedSettings["clusterSubnetLength"].(int)),
        "machineCIDR": parsedSettings["machineCIDR"].(string),
        "serviceCIDR": parsedSettings["serviceCIDR"].(string),
        "SDNType": parsedSettings["SDNType"].(string),
    }
    // Settings depending on provider
    if _, ok := parsedSettings["libvirtURI"]; ok {
        settings["libvirtURI"] = parsedSettings["libvirtURI"].(string)
    }

    // Merge with secrets dictionary
    for k, v := range g.secrets {
        settings[k] = v
    }
    err = t.Execute(f, settings)

    if err != nil {
        log.Fatal(fmt.Sprintf("Error parsing template: %s", err))
    }

}

func (g generator) CreateManifests() {
    cmd := exec.Command("openshift-install", "create", "manifests")
    cmd.Dir = g.buildPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error creating manifests: %s - %s", err, string(out)))
        os.Exit(1)
    }
}

func (g generator) DeployCluster() {
    cmd := exec.Command("openshift-install", "create", "cluster")
    cmd.Dir = g.buildPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        log.Fatal(fmt.Sprintf("Error creating cluster: %s - %s", err, string(out)))
        os.Exit(1)
    }
}

func (g generator) GenerateManifests() {
    // First download the needed artifacts
    g.DownloadArtifacts()

    // Generate install-config.yaml
    g.GenerateInstallConfig()

    // Create manifests
    g.CreateManifests()

    // Deploy cluster
    g.DeployCluster()
}
