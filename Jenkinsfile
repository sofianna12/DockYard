pipeline {
    agent any

    stages {
        stage('Build & Push Frontend') {
            steps {
                script {
                    docker.withRegistry('https://index.docker.io/v1/', 'dockerhub-credentials') {
                        def img = docker.build("asofia32/dockyard-frontend", "./frontend")
                        img.push("latest")
                        img.push("${env.GIT_COMMIT}")
                    }
                }
            }
        }
        stage('Build & Push Backend') {
            steps {
                script {
                    docker.withRegistry('https://index.docker.io/v1/', 'dockerhub-credentials') {
                        def img = docker.build("asofia32/dockyard-backend", "./backend")
                        img.push("latest")
                        img.push("${env.GIT_COMMIT}")
                    }
                }
            }
        }
    }
}
