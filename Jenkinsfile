rrpBuildGoCode {
    projectKey = 'product-data-service'
    testDependencies = ['postgresql']
    dockerBuildOptions = ['--squash', '--build-arg GIT_COMMIT=$GIT_COMMIT']
    ecrRegistry = "280211473891.dkr.ecr.us-west-2.amazonaws.com"
    buildImage = 'amr-registry.caas.intel.com/rrp/ci-go-build-image:1.12.0-alpine-postgres'
    dockerImageName = "rrs/${projectKey}"
    protexProjectName = 'bb-product-data-service'
    customBuildScript = "./build.sh"

    infra = [
        stackName: 'RSP-Codepipeline-ProductDataService'
    ]

    notify = [
        slack: [ success: '#ima-build-success', failure: '#ima-build-failed' ]
    ]
}
