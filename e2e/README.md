 
## The intent of these changes

1. Separate infrastructure setup from test runs
2. Organize configuration parameters by infra, provider, and provisioners
3. Introduce a structure for defining new test parameters if needed.
5. Evaluate test requirements and validation before running tests. Make it dynamic, so only targeted tests will be validated.
fd
6. Allow nested tests to reuse parent parameters.

Example:
```
 roboe2e -grv-init=local 
 roboe2e -grv-tests=tests.installer.onprem 
 roboe2e -grv-tests=tests.cluster.expand.onprem 

  or

 roboe2e -grv-tests=tests.installer.*
```


## Example of config file

```yaml
# runtime
report_dir: /tmp/robotest-reports
login:
    username: xxxx
    password: xxxx
    auth_provider: google
infras:
    remote:        
        name: alexey-infra
        url: https://portal.gravitational.io/
    local:
        name: alexey-infra
        provisioner: vagrant
        tarball_path: /home/akontsevoy/grv-apps/alex/installer.tar.gz
providers:
    aws:
        access_key: XXXXX
        secret_key: XXXXXX
        region: xxxx
        key_path: /path/to/SSH/key
        key_pair: xxx
        ssh_user: ubuntu
provisioners:
    terraform:      
        nodes: 3
        script_path: xxxxx        
        provider: aws        
    vagrant:                
        nodes: 2    
        script_path: "/home/akontsevoy/go/src/github.com/gravitational/robotest/assets/vagrant/Vagrantfile"                
        docker_device: /dev/sdd

# installer tests
tests.installer:
    cluster_name: xxxx  
    license: "application license"            
tests.installer.onprem:        
    docker_device: /dev/sdd
    flavor_label: "1 node"    
tests.installer.aws:    
    app: gravitational.io/assembla-private-cloud:0.1.1-beta-2-g3066484            
    region: us-west-1
    key_pair: ops
    vpc: "vpc-6fbb610a (default)"    
    instance_type: "m3.xlarge"
    flavor_label: "node 1"    

# cluster tests
tests.cluster:      
    cluster_name: xxxxx 
tests.cluster.expand.onprem:        
    profile: node    
tests.cluster.expand.aws:        
    profile: node
    instance_type: "m3.xlarge"        
```