// 返回跨平台的环境变量设置命令
def setupBuildEnv() {
    if (isUnix()) {
        return '''
            export PATH="${HOME}/go/bin:${PYTHON_INSTALL_DIR}/bin:${GO_INSTALL_DIR}/bin:${PATH}"
            export BUILD_VERSION="${BUILD_VERSION}"
            export CONFIG_VERSION="${CONFIG_VERSION}"
            export BUILD_MODE="${BUILD_MODE}"
            export GIT_BRANCH="${GIT_BRANCH}"
            export GOTRACEBACK=crash
        '''
    } else {
        return '''
            set "PATH=${HOME}\\go\\bin;${PYTHON_INSTALL_DIR}\\bin;${GO_INSTALL_DIR}\\bin;%PATH%"
            set "BUILD_VERSION=${BUILD_VERSION}"
            set "CONFIG_VERSION=${CONFIG_VERSION}"
            set "BUILD_MODE=${BUILD_MODE}"
            set "GIT_BRANCH=${GIT_BRANCH}"
            set "GOTRACEBACK=crash"
        '''
    }
}

def cleanWorkspace() {
    if (isUnix()) {
        echo 'Cleaning workspace (Linux/Unix)...'
        sh """
            rm -rf "${BUILD_TOOLS_DIR}" \\
                   ./.venv \\
                   ./.cache \\
                   ./build \\
                   ./.pytest_cache
            echo "✓ Workspace cleaned"
        """
    } else {
        echo 'Cleaning workspace (Windows)...'
        powershell """
            Remove-Item -Recurse -Force -ErrorAction SilentlyContinue `
                '${BUILD_TOOLS_DIR}', `
                './.venv', `
                './.cache', `
                './build', `
                './.pytest_cache'
            Write-Host '✓ Workspace cleaned'
        """
    }
}

pipeline {
    agent {
        label "${params.NODE_LABEL ?: 'linux'}"
    }

    parameters {
        booleanParam(name: 'FULL_BUILD', defaultValue: false, description: 'Clean workspace before build')
        choice(name: 'NODE_LABEL', choices: ['linux', 'windows'], description: '选择构建平台')
        string(name: 'BUILD_VERSION', defaultValue: 'v1.0.0', description: '构建版本号 (如: v1.0.0, v1.0.0-rc.1)')
        string(name: 'CONFIG_VERSION', defaultValue: 'v1.0.0', description: '配置版本号 (如: v1.0.0)')
        string(name: 'GIT_BRANCH', defaultValue: 'main', description: '服务器分支')
        choice(name: 'BUILD_MODE', choices: ['Debug', 'Release'], description: '构建模式')
    }

    environment {
        GO_VERSION = '1.27.4'
        // 归档相关
        LINUX_IP = '10.64.5.1'
        LINUX_USER = 'Prx_UGS'
        BUILD_TIME = "${new Date().format('yyyyMMddHHmm', TimeZone.getTimeZone('Asia/Shanghai'))}"
    }

    stages {

        stage('Initialize') {
            steps {
                script {
                    env.BUILD_TOOLS_DIR = "${env.WORKSPACE}/.tools"
                    env.GO_INSTALL_DIR = "${env.BUILD_TOOLS_DIR}/go"
                    env.PYTHON_INSTALL_DIR = "${env.BUILD_TOOLS_DIR}/python"
                    echo "Workspace: ${env.WORKSPACE}"
                }
            }
        }

        stage('Clean Workspace') {
            when {
                expression { params.FULL_BUILD == true }
            }
            steps {
                script {
                    cleanWorkspace()
                }
            }
        }

        stage('Check & Prepare Toolchain') {
            steps {
                echo 'Checking Go and Python/uv build environment...'
                script {
                    def needInstallGo = true
                    def goExists = sh(script: 'command -v go', returnStatus: true) == 0
                    if (goExists) {
                        def goVersion = sh(script: 'go version', returnStdout: true).trim()
                        echo "Go is installed: ${goVersion}"
                        def versionMatch = sh(script: "go version | grep -q 'go${GO_VERSION}'", returnStatus: true) == 0
                        if (versionMatch) {
                            echo "✓ Go version matches required version: ${GO_VERSION}"
                            needInstallGo = false
                        } else {
                            echo "✗ Go version mismatch. Required: ${GO_VERSION}"
                        }
                    }

                    if (needInstallGo && isUnix()) {
                        echo "Installing Go ${GO_VERSION} in workspace..."
                        sh '''
                            mkdir -p ${GO_INSTALL_DIR}
                            cd ${WORKSPACE}/.tools
                            if [ ! -f "go${GO_VERSION}.linux-amd64.tar.gz" ]; then
                                curl -fsSLO https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
                            fi
                            tar -xzf go${GO_VERSION}.linux-amd64.tar.gz
                            ${GO_INSTALL_DIR}/bin/go version
                        '''
                    }

                    sh """${setupBuildEnv()}
                        git submodule update --init --recursive || git submodule update --init --recursive
                        if command -v mise >/dev/null 2>&1; then
                            mise install --yes
                        fi
                        go tool dev doctor
                    """
                }
            }
        }

        stage('Build & Codegen') {
            steps {
                script {
                    def forceFlag = params.FULL_BUILD ? '-force-codegen' : ''
                    sh """${setupBuildEnv()}
                        go tool dev build -mode "${params.BUILD_MODE}" -version "${params.BUILD_VERSION}" -config-version "${params.CONFIG_VERSION}" -branch "${params.GIT_BRANCH}" ${forceFlag}
                        if [[ "${params.GIT_BRANCH}" == "main" ]] || [[ "${params.GIT_BRANCH}" == "origin/main" ]]; then
                            go tool dev docs
                        fi
                    """
                }
            }
            post {
                success {
                    echo 'Build completed successfully!'
                }
                failure {
                    echo 'Build failed! Please check the logs.'
                }
            }
        }

        stage('Verify Standalone Submodules') {
            steps {
                sh """${setupBuildEnv()}
                    go tool dev verify-standalone -test
                """
            }
        }

        stage('Run Tests') {
            steps {
                echo 'Running tests in atsf4g-go...'
                sh """${setupBuildEnv()}
                    go tool dev test -v -timeout 10m
                """
            }
        }

        stage('Run CI Lint') {
            steps {
                echo 'Running golangci-lint checks (including depguard visibility rules)...'
                sh """${setupBuildEnv()}
                    if command -v golangci-lint >/dev/null 2>&1; then
                        go tool dev lint
                    else
                        echo "⚠ golangci-lint not found on agent PATH; skipping lint stage"
                    fi
                """
            }
        }

        stage('生成配置') {
            when {
                expression { env.NODE_LABEL == 'windows' }
            }
            steps {
                script {
                    bat """
                        cd /d "${env.WORKSPACE}\\build\\install"
                        call update_dependency.bat
                        call generate_config.bat
                    """
                }
            }
        }

        stage('打包构建产物') {
            steps {
                script {
                    echo "开始打包构建产物..."
                    def sourceDir = "${env.WORKSPACE}/build/install"
                    def branchName = params.GIT_BRANCH.contains('/') ? params.GIT_BRANCH.split('/').last() : params.GIT_BRANCH
                    def zipFile = "ProjectY_Server_${branchName}_${env.BUILD_NUMBER}.tar.gz"
                    def docTarFile = "ProjectY_Server_Doc.tar.gz"
                    if (params.NODE_LABEL != 'linux') {
                        env.ARCHIVE_PATH = "/data/archive/disk1/nextcloud/temporary/ProjectY/Server/${params.NODE_LABEL}/${env.BUILD_TIME}_${env.BUILD_NUMBER}"
                        env.DOCUMENT_PATH = "/data/archive/disk1/nextcloud/temporary/ProjectY/Server/Document"
                    } else {
                        env.ARCHIVE_PATH = "/data/archive/disk1/nextcloud/temporary/ProjectY/Server/${params.NODE_LABEL}"
                        env.DOCUMENT_PATH = "/data/archive/disk1/nextcloud/temporary/ProjectY/Server/Document"
                    }
                    sh """
                        tar -czvf "${zipFile}" -C "${sourceDir}" .
                        if [[ "${params.GIT_BRANCH}" == "main" ]] || [[ "${params.GIT_BRANCH}" == "origin/main" ]]; then
                            tar -czvf "${docTarFile}" -C "${env.WORKSPACE}/doc/site" .
                        fi
                    """
                    echo "打包完成: ${zipFile}"

                    echo "开始通过SCP传输文件到Linux服务器..."
                    sh """
                        ssh -p 36000 -o StrictHostKeyChecking=no ${env.LINUX_USER}@${env.LINUX_IP} "mkdir -p ${env.ARCHIVE_PATH} ${env.DOCUMENT_PATH}"
                        scp -P 36000 -o StrictHostKeyChecking=no "${zipFile}" ${env.LINUX_USER}@${env.LINUX_IP}:${env.ARCHIVE_PATH}/
                        ssh -p 36000 -o StrictHostKeyChecking=no ${env.LINUX_USER}@${env.LINUX_IP} "find ${env.ARCHIVE_PATH} -mindepth 1 -maxdepth 1 -mtime +5 -exec rm -rf {} +"
                        ssh -p 36000 -o StrictHostKeyChecking=no ${env.LINUX_USER}@${env.LINUX_IP} 'count=\$(find '"${env.ARCHIVE_PATH}"' -mindepth 1 -maxdepth 1 -type d | wc -l); if [ \$count -gt 20 ]; then find '"${env.ARCHIVE_PATH}"' -mindepth 1 -maxdepth 1 -type d -printf ''%T@ %p\\n'' | sort -rn | tail -n +21 | cut -d'' '' -f2- | xargs -r rm -rf; fi'
                        if [[ "${params.GIT_BRANCH}" == "main" ]] || [[ "${params.GIT_BRANCH}" == "origin/main" ]]; then
                            scp -P 36000 -o StrictHostKeyChecking=no "${docTarFile}" ${env.LINUX_USER}@${env.LINUX_IP}:${env.DOCUMENT_PATH}/
                            ssh -p 36000 -o StrictHostKeyChecking=no ${env.LINUX_USER}@${env.LINUX_IP} "cd ${env.DOCUMENT_PATH} && tar -xzvf ProjectY_Server_Doc.tar.gz"
                        fi
                    """
                    echo "文件传输成功完成。"
                    if (isUnix()) {
                        sh """
                            rm -f "${zipFile}"
                        """
                    } else {
                        powershell """
                            Remove-Item -Path "${zipFile}" -Force -ErrorAction SilentlyContinue
                        """
                    }

                    def meta = [package_name: "${zipFile}",
                                archive_path: "${env.ARCHIVE_PATH}"
                                ]
                    currentBuild.displayName = "package_data"
                    currentBuild.description = groovy.json.JsonOutput.toJson(meta)
                }
            }
        }
    }

    post {
        success {
            script {
                if (params.NODE_LABEL != 'linux') {
                    echo '✅ Pipeline completed successfully!'

                    writeFile file: 'payload.json', text: """
                    {
                      "msg_type": "post",
                      "content": {
                        "post": {
                          "zh_cn": {
                            "title": "服务器${env.NODE_LABEL}包构建成功",
                            "content": [
                              [
                                {
                                  "tag": "text",
                                  "text": "构建号:${env.BUILD_NUMBER}\\n"
                                },
                                {
                                  "tag": "a",
                                  "text": "下载连接",
                                  "href": "https://nextcloud.m-oa.com:6023/apps/files/files?dir=/%E4%B8%B4%E6%97%B6%E5%85%B1%E4%BA%AB%28temporary%29/ProjectY/Server/${params.NODE_LABEL}/${env.BUILD_TIME}_${env.BUILD_NUMBER}"
                                }
                              ]
                            ]
                          }
                        }
                      }
                    }
                    """
                    sh 'curl -sS -X POST -H "Content-Type: application/json" -d "@payload.json" "$FEISHU_PROJECT_Y_URL"'
                } else {
                    echo '✅ Pipeline completed successfully! (Skipping notification for non-windows build)'
                }
            }
        }
        failure {
            echo '❌ Pipeline failed!'
            script {
                writeFile file: 'payload.json', text: """
                {
                  "msg_type": "post",
                  "content": {
                    "post": {
                      "zh_cn": {
                        "title": "服务器${env.NODE_LABEL}构建失败",
                        "content": [
                          [
                            {
                              "tag": "text",
                              "text": "构建号:${env.BUILD_NUMBER}\\n构建平台:${params.NODE_LABEL}"
                            },
                            {
                              "tag": "a",
                              "text": "构建连接",
                              "href": "https://jenkins.m-oa.com:6023/job/ProjectY/job/Server/job/Server_Build/${env.BUILD_NUMBER}/console"
                            }
                          ]
                        ]
                      }
                    }
                  }
                }
                """
                sh 'curl -sS -X POST -H "Content-Type: application/json" -d "@payload.json" "$FEISHU_PROJECT_Y_URL"'
            }
        }
    }
}
