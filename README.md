## KNI installer

This repository contains the installer for the Akraino KNI deployment. Along with the [https://gerrit.akraino.org/r/#/admin/projects/kni/templates](https://gerrit.akraino.org/r/#/admin/projects/kni/templates) repository, it will allow to deploy Edge sites in a declarative way.

## Dependencies

You will need to create an account on [http://cloud.openshift.com/](http://cloud.openshift.com/)
This is needed to have download access to the OpenShift installer artifacts.
After that, you will need to download the Pull Secret from
[https://cloud.openshift.com/clusters/install](https://cloud.openshift.com/clusters/install) - Step 4: Deploy the Cluster

## How to build

First the `kni-edge-installer` binary needs to be produced. For that you just execute make with the following syntax:

    make build

This will produce the `kni-edge-installer` binary that can be used to deploy a site.

## How to deploy

There is a Makefile on the root directory of this project. In order to deploy
you will need to use the following syntax:

    make deploy CREDENTIALS=<git_private_repo> SETTINGS=<path_to_sample_settings> BASE_REPO=<git_base_manifests_repo> BASE_PATH=<path_in_manifests_repo>

**CREDENTIALS: Content of private repo**

This repository is needed to store private credentials. It is recommended that
you store those credentials on a private repo where only allowed people have
access. Each setting is stored in individual files in the repository:

- ssh-pub-key           # public key to be used for SSH access into the nodes
- coreos-pull-secret    # place the file that you created before
- aws-access-key-id     # just for AWS deploy, key id that you created in AWS
- aws-secret-access-key # just for AWS deploy, secret for the key id

The right path to clone a private repo is: git@github.com:repo_user/repo_name.git

**BASE_REPO: Repository for the base manifests**

This is the repository where the default manifest templates are stored. There is one specific folder for each blueprint and provider: aws/1-node, libvirt/1-node, etc... This can be any repository with the right templates, but for Akraino it currently defaults to github.com/redhat-nfvpe/kni-edge-base.git

**BASE_PATH: Path inside base manifests**

Once the manifests repository is being cloned, it will contain several folders with all the specific manifests for the blueprint. In order to choose a provider and footprint, the BASE_PATH needs to be specific. Current values are: libvirt/1-node, libvirt/3-node, aws/1-node, aws/3-node

**SETTINGS: Specific site settings**

This is going to contain the settings that will be different per deployment. Depending on the provider it will contain some specific settings. A sample settings file can be seen at: [https://raw.githubusercontent.com/redhat-nfvpe/kni-edge-base/master/libvirt/sample_settings.yaml](https://raw.githubusercontent.com/redhat-nfvpe/kni-edge-base/master/libvirt/sample_settings.yaml ) . But different settings per site can be created on different git repos.

## How to deploy for AWS

Before starting the deploy, please read the following documentation to prepare
the AWS account properly:
[https://github.com/openshift/installer/blob/master/docs/user/aws/README.md](https://github.com/openshift/installer/blob/master/docs/user/aws/README.md)

There are two different footprints for AWS: 1 master/1 worker, and 3 masters/3 workers. Makefile needs to be called with:

    make deploy CREDENTIALS=git@github.com:repo_user/repo_name.git SETTINGS=https://raw.githubusercontent.com/redhat-nfvpe/kni-edge-base/master/aws/sample_settings.yaml BASE_REPO=github.com/redhat-nfvpe/kni-edge-base.git BASE_PATH=[aws/3-node|aws/1-node]


The file will look like:

    settings:
      baseDomain: "<base_domain>"
      clusterName: "<cluster_name>"
      clusterCIDR: "10.128.0.0/14"
      clusterSubnetLength: 9
      machineCIDR: "10.0.0.0/16"
      serviceCIDR: "172.30.0.0/16"
      SDNType: "OpenShiftSDN"
      AWSRegion: "<aws_region_to_deploy>"

Where:
- `<base_domain>` is the DNS zone matching with the one created on Route53
- `<cluster_name>` is the name you are going to give to the cluster
- `<aws_region_to_deploy>` is the region where you want your cluster to deploy

SETTINGS can be a path to local file, or an url, will be queried with curl.

The make process will create the needed artifacts and will start the deployment of the specified cluster

## How to deploy for Libvirt

First of all, we need to prepare a host in order to configure libvirt, iptables, permissions, etc. So far this is a manual process:

[https://github.com/openshift/installer/blob/master/docs/dev/libvirt-howto.md](https://github.com/openshift/installer/blob/master/docs/dev/libvirt-howto.md)

Unfortunately, Libvirt is only for development purposes from the OpenShift perspective, so the binary is not compiled with the libvirt bits by default. The user will have to compile it by his/her own version with libvirt enabled.
The link pasted above, also contains the instructions to compile the installer with the correct tags. Once it is compiled correctly, you will have to point to the binary from the execution command (make).

There are two different footprints for libvirt: 1 master/1 worker, and 3 masters/3 workers. Makefile needs to be called with:

    make deploy CREDENTIALS=git@github.com:repo_user/repo_name.git SETTINGS=https://raw.githubusercontent.com/redhat-nfvpe/kni-edge-base/master/libvirt/sample_settings.yaml BASE_REPO=github.com/redhat-nfvpe/kni-edge-base.git BASE_PATH=[libvirt/3-node|libvirt/1-node] INSTALLER_PATH=file:///${GOPATH}/bin/openshift-install

A sample settings.yaml file has been created specifically for Libvirt targets. It needs to look like:

    settings:
      baseDomain: "<base_domain>"
      clusterName: "<cluster_name>"
      clusterCIDR: "10.128.0.0/14"
      clusterSubnetLength: 9
      machineCIDR: "10.0.0.0/16"
      serviceCIDR: "172.30.0.0/16"
      SDNType: "OpenShiftSDN"
      libvirtURI: "<libvirt_host_ip>"

Where:
- `<base_domain>` is the DNS zone matching with the entry created in /etc/NetworkManager/dnsmasq.d/openshift.conf during the libvirt-howto machine setup. (tt.testing by default)
- `<cluster_name>` is the name you are going to give to the cluster
- `<libvirt_host_ip>` is the host IP where libvirt is configured (i.e. qemu+tcp://192.168.122.1/system)

The rest of the options are exactly the same as in an AWS deployment.

**Temporary workaround**

Currently the installer is failing when adding console to the cluster for libvirt. In order to make it work, please follow instructions on [https://github.com/openshift/installer/pull/1371](https://github.com/openshift/installer/pull/1371).

## How to use the cluster

After the deployment finishes, a `kubeconfig` file will be placed inside
build/auth directory:

    export KUBECONFIG=./build/auth/kubeconfig

Then cluster can be managed with oc. You can get the client on this link
[https://cloud.openshift.com/clusters/install](https://cloud.openshift.com/clusters/install)
- Step 5: Access your new cluster.

## How to destroy the cluster

In order to destroy the running cluster, and clean up environment, just use
`make clean` command.

## Building and consuming your own installer

The openshift-installer binaries are published on [https://github.com/openshift/installer/releases](https://github.com/openshift/installer/releases).
For faster deploy, you can grab the installer from here. However, there may be situations where you need to compile your own installer (such as the case of libvirt), or you need a newer version.
In that case, you can build it following the instructions on [https://github.com/openshift/installer](https://github.com/openshift/installer)

Then you can export the path to the new installer before running make:

    export INSTALLER_PATH=http://<url_to_binary>/openshift-install

Or pass it as a parameter to make command.

## Customization: use your own manifests

openshift-installer is also able to produce manifests, that end users can modify
and deploy a cluster with the modified settings. New manifests can be generated
with:

    /path/to/openshift-install create manifests

This will generate a pair of folders: manifests and openshift. Those manifests
can be modified with the desired values. After that this code can be executed to
generate a new cluster based on the modified manifests:

    /path/to/openshift-install create cluster
