## XBeam Workload Infrastructure as a Code

XBeam Workload IaaC enables you to run XBeam infrastructure with no knowledge.

### Steps to run XBeam Workload IaaC
1. Install Pulumi for your OS 
2. Clone the XBeam Workload IaaC repository
3. Update the `Pulumi.dev.yaml` based on your requirements  
    3.1. Make sure you have changed the windows worker password.  
    3.2. Make sure you have configured the AWS cli with the correct credentials.
4. Run `pulumi up --config-file Pulumi.dev.yaml` to create the infrastructure
* Run `pulumi destroy --config-file Pulumi.dev.yaml` to destroy the infrastructure

the UP and Destroy commands will take 20-30 minutes to complete.  

Your XBeam infrastructure is now ready to get configured by the XBeam Installer.
