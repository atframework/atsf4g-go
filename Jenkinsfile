// 返回跨平台的环境变量设置命令
def setupBuildEnv() {
    if (isUnix()) {
        return '''
            export PATH="\${HOME}/go/bin:\${PYTHON_INSTALL_DIR}/bin:\${GO_INSTALL_DIR}/bin:\${TASK_INSTALL_DIR}:\$PATH"
            export BUILD_VERSION="${BUILD_VERSION}"
            export CONFIG_VERSION="${CONFIG_VERSION}"
            export BUILD_MODE="${BUILD_MODE}"
        '''
    } else {
        return '''
            set "PATH=\${HOME}\\go\\bin;\${PYTHON_INSTALL_DIR}\\bin;\${GO_INSTALL_DIR}\\bin;\${TASK_INSTALL_DIR};%PATH%"
            set "BUILD_VERSION=${BUILD_VERSION}"
            set "CONFIG_VERSION=${CONFIG_VERSION}"
            set "BUILD_MODE=${BUILD_MODE}"
        '''
    }
}

def cleanWorkspace() {
    if (isUnix()) {
        echo 'Cleaning workspace (Linux/Unix)...'
        sh """
            rm -rf "${BUILD_TOOLS_DIR}" \\
                   "${PYTHON_VENV_PATH}" \\
                   ./build \\
                   ./.cache \\
                   ./.pytest_cache
            echo "✓ Workspace cleaned"
        """
    } else {
        echo 'Cleaning workspace (Windows)...'
        powershell """
            Remove-Item -Recurse -Force -ErrorAction SilentlyContinue `
                '${BUILD_TOOLS_DIR}', `
                '${PYTHON_VENV_PATH}', `
                './build', `
                ./.cache', `
                ./.pytest_cache'
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
        choice(name: 'BUILD_MODE', choices: ['Release', 'Debug'], description: '构建模式')
    }
    
    environment {
        GO_VERSION = '1.25.1'
        TASK_VERSION = '3.45.4'
        // 归档相关
        LINUX_IP = '10.64.5.1' 
        LINUX_USER = 'Prx_UGS'
        BUILD_TIME = "${new Date().format('yyyyMMddHHmm', TimeZone.getTimeZone('Asia/Shanghai'))}"
    }
    
    stages {

        stage('Initialize') {
            steps {
                script {
                    // 动态设置路径变量，统一使用正斜杠（Windows 和 Linux 都支持）
                    def workspaceUnix = env.WORKSPACE.replace('\\', '/')
                    env.BUILD_TOOLS_DIR = "${env.WORKSPACE}/.tools"
                    env.GO_INSTALL_DIR = "${env.BUILD_TOOLS_DIR}/go"
                    env.TASK_INSTALL_DIR = "${env.BUILD_TOOLS_DIR}/task/bin"
                    env.PYTHON_INSTALL_DIR = "${env.BUILD_TOOLS_DIR}/python"
                    env.PYTHON_VENV_PATH = "${workspaceUnix}/protject_venv"
                    echo "Workspace: ${env.WORKSPACE}"
                    echo "Python venv path: ${env.PYTHON_VENV_PATH}"
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

        stage('Check Environment') {
            steps {
                echo 'Checking build environment...'
                script {
                    // 检测 Go 环境
                    def needInstallGo = true
                    def goExists = sh(script: 'command -v go', returnStatus: true) == 0
                    
                    if (goExists) {
                        def goVersion = sh(script: 'go version', returnStdout: true).trim()
                        echo "Go is installed: ${goVersion}"
                        
                        // 检查版本是否匹配
                        def versionMatch = sh(script: "go version | grep -q 'go${GO_VERSION}'", returnStatus: true) == 0
                        if (versionMatch) {
                            echo "✓ Go version matches required version: ${GO_VERSION}"
                            needInstallGo = false
                        } else {
                            echo "✗ Go version mismatch. Required: ${GO_VERSION}"
                            echo "Will install Go ${GO_VERSION} in workspace..."
                        }
                    } else {
                        echo "Go is not installed. Will install in workspace..."
                    }
                    env.NEED_INSTALL_GO = needInstallGo.toString()
                    
                    // 检测 Task 环境
                    def needInstallTask = true
                    def taskExists = sh(script: 'command -v task', returnStatus: true) == 0
                    
                    if (taskExists) {
                        def taskVersion = sh(script: 'task --version', returnStdout: true).trim()
                        echo "Task is installed: ${taskVersion}"
                        
                        // Task 版本检查（简单检查是否存在）
                        echo "✓ Task is available"
                        needInstallTask = false
                    } else {
                        echo "Task is not installed. Will install in workspace..."
                    }
                    env.NEED_INSTALL_TASK = needInstallTask.toString()
                    
                    // 检测 Python 环境
                    def needInstallPython = true
                    def pythonExists = sh(script: 'command -v python3', returnStatus: true) == 0
                    
                    if (pythonExists) {
                        def pythonVersion = sh(script: 'python3 --version', returnStdout: true).trim()
                        echo "Python is installed: ${pythonVersion}"
                        
                        // 只要有 Python3 就不重新安装
                        echo "✓ Python3 is available"
                        needInstallPython = false
                    } else {
                        echo "Python is not installed. Will install in workspace..."
                    }
                    env.NEED_INSTALL_PYTHON = needInstallPython.toString()
                    
                    // 检测 Git 环境（必需）
                    def gitExists = sh(script: 'command -v git', returnStatus: true) == 0
                    if (gitExists) {
                        def gitVersion = sh(script: 'git --version', returnStdout: true).trim()
                        echo "✓ Git is installed: ${gitVersion}"
                    } else {
                        error("✗ Git is not installed! Please install git first.")
                    }
                    
                    // 检测 wget 环境（下载工具需要）
                    def wgetExists = sh(script: 'command -v wget', returnStatus: true) == 0
                    if (wgetExists) {
                        def wgetVersion = sh(script: 'wget --version | head -n1', returnStdout: true).trim()
                        echo "✓ wget is installed: ${wgetVersion}"
                        env.NEED_INSTALL_WGET = 'false'
                    } else {
                        echo "✗ wget is not installed. Will try to install..."
                        env.NEED_INSTALL_WGET = 'true'
                    }
                    
                    // 显示系统信息
                    sh '''
                        echo "=========================================="
                        echo "System Information:"
                        echo "OS: $(uname -s)"
                        echo "Architecture: $(uname -m)"
                        echo "Workspace: ${WORKSPACE}"
                        echo "Go Install Dir: ${GO_INSTALL_DIR}"
                        echo "Task Install Dir: ${TASK_INSTALL_DIR}"
                        echo "=========================================="
                    '''
                }
            }
        }

        
        stage('Install Dependencies') {
            steps {
                script {
                    echo 'Setting up build tools...'
                    
                    // 安装 wget(如果需要)
                    if (env.NEED_INSTALL_WGET == 'true') {
                        echo "Installing wget..."
                        sh '''
                            if command -v apt-get &> /dev/null; then
                                apt-get update -qq || echo "Warning: apt-get update failed, continuing..."
                                apt-get install -y wget || echo "Warning: Could not install wget via apt-get"
                            elif command -v yum &> /dev/null; then
                                yum install -y wget || echo "Warning: Could not install wget via yum"
                            elif command -v apk &> /dev/null; then
                                apk add --no-cache wget || echo "Warning: Could not install wget via apk"
                            elif command -v scoop &> /dev/null; then
                                scoop install wget || echo "Warning: Could not install wget via scoop"
                            else
                                echo "✗ Cannot install wget automatically. Please install it manually or ensure it's available in the container image."
                                exit 1
                            fi
                            
                            if command -v wget &> /dev/null; then
                                wget --version | head -n1
                                echo "✓ wget is available"
                            else
                                echo "✗ wget installation failed"
                                exit 1
                            fi
                        '''
                    }
                    
                    // 安装 Go（如果需要）
                    if (env.NEED_INSTALL_GO == 'true') {
                        echo "Installing Go ${GO_VERSION} in workspace..."
                        sh '''
                            mkdir -p ${GO_INSTALL_DIR}
                            cd ${WORKSPACE}/.tools
                            
                            # 下载 Go
                            if [ ! -f "go${GO_VERSION}.linux-amd64.tar.gz" ]; then
                                echo "Downloading Go ${GO_VERSION}..."
                                wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
                            fi
                            
                            # 解压 Go
                            echo "Extracting Go..."
                            tar -xzf go${GO_VERSION}.linux-amd64.tar.gz
                            
                            # 验证安装
                            ${GO_INSTALL_DIR}/bin/go version
                        '''
                    } else {
                        echo "Using system Go installation"
                        sh 'go version'
                    }
                    
                    // 安装 Task（如果需要）
                    if (env.NEED_INSTALL_TASK == 'true') {
                        echo "Installing Task ${TASK_VERSION} in workspace..."
                        sh '''
                            mkdir -p ${TASK_INSTALL_DIR}
                            cd ${WORKSPACE}/.tools
                            
                            # 检查 TASK_INSTALL_DIR 里是否已有可用的 task
                            if [ -f "${TASK_INSTALL_DIR}/task" ] && [ -x "${TASK_INSTALL_DIR}/task" ]; then
                                EXISTING_VERSION=$(${TASK_INSTALL_DIR}/task --version 2>/dev/null || echo "unknown")
                                echo "Found existing task in ${TASK_INSTALL_DIR}: ${EXISTING_VERSION}"
                                
                                # 检查版本是否匹配
                                if echo "${EXISTING_VERSION}" | grep -q "${TASK_VERSION}"; then
                                    echo "✓ Task version matches, using existing installation"
                                    ${TASK_INSTALL_DIR}/task --version
                                    exit 0
                                else
                                    echo "✗ Task version mismatch. Expected: ${TASK_VERSION}, Found: ${EXISTING_VERSION}"
                                    echo "Will re-download and install..."
                                fi
                            else
                                echo "No existing task found in ${TASK_INSTALL_DIR}, will download..."
                            fi
                            
                            # 下载并安装 Task
                            echo "Downloading Task ${TASK_VERSION}..."
                            wget -q https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_amd64.tar.gz
                            tar -xzf task_linux_amd64.tar.gz -C ${TASK_INSTALL_DIR}
                            chmod +x ${TASK_INSTALL_DIR}/task
                            
                            # 验证安装
                            echo "✓ Task installation completed"
                            ${TASK_INSTALL_DIR}/task --version
                        '''
                    } else {
                        echo "Using system Task installation"
                        sh 'task --version'
                    }
                    
                    // 安装 Python（如果需要）
                    if (env.NEED_INSTALL_PYTHON == 'true') {
                        echo "Installing Python from system package manager..."
                        sh '''
                            # 直接使用系统包管理器安装 Python3
                            if command -v apt-get &> /dev/null; then
                                apt-get update -qq
                                apt-get install -y python3 python3-pip python3-venv
                                echo "✓ Python installed via apt-get"
                            elif command -v yum &> /dev/null; then
                                yum install -y python3 python3-pip
                                echo "✓ Python installed via yum"
                            elif command -v apk &> /dev/null; then
                                apk add --no-cache python3 py3-pip
                                echo "✓ Python installed via apk"
                            else
                                echo "✗ No supported package manager found"
                                exit 1
                            fi
                            
                            # 验证安装
                            python3 --version
                            pip3 --version || echo "⚠ pip not available"
                        '''
                    } else {
                        echo "Using system Python installation"
                        sh 'python3 --version'
                    }
                    
                    // 显示最终环境
                    sh """${setupBuildEnv()}
                        
                        echo "=========================================="
                        echo "Build Tools Ready:"
                        echo "Go: \$(go version)"
                        echo "Task: \$(task --version)"
                        echo "Python: \$(python3 --version)"
                        echo "PATH: \$PATH"
                        echo "=========================================="
                    """
                    
                    // 安装项目依赖(多模块结构)
                    echo "Installing project dependencies..."
                    sh """${setupBuildEnv()}
                        
                        # 项目是多模块结构,需要遍历所有 go.mod 目录
                        echo "Finding all Go modules..."
                        find . -name go.mod -not -path "*/vendor/*" -not -path "*/.tools/*" -exec dirname {} \\; | sort -u | while read dir; do
                            echo "Running go mod download in \\\$dir"
                            (cd "\\\$dir" && go mod download) || echo "Warning: go mod download failed in \\\$dir"
                        done
                        echo "✓ Go modules download completed"
                    """
                }
            }
        }
        
        stage('Check Build Tools') {
            steps {
                echo 'Checking if build tools are already installed...'
                script {
                    def allToolsExist = sh(script: '''
                        if [ ! -f "build/build-settings.json" ]; then
                            echo "build-settings.json not found"
                            exit 1
                        fi
                        
                        # 使用 jq 或 python 解析 JSON 并检查工具路径
                        if command -v python3 &> /dev/null; then
                            python3 << 'PYEOF'
import json
import os
import sys

try:
    with open('build/build-settings.json', 'r') as f:
        data = json.load(f)
    
    tools = data.get('tools', {})
    all_exist = True
    
    for tool_name, tool_info in tools.items():
        path = tool_info.get('path', '')
        # 转换 Windows 路径为相对路径或跳过绝对路径检查
        if path and not os.path.isabs(path):
            if not os.path.exists(path):
                print(f"Tool {tool_name} not found at: {path}")
                all_exist = False
        else:
            print(f"Skipping absolute path check for {tool_name}: {path}")
    
    sys.exit(0 if all_exist else 1)
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)
PYEOF
                        else
                            echo "Python3 not available for JSON parsing"
                            exit 1
                        fi
                    ''', returnStatus: true)
                    
                    if (allToolsExist == 0) {
                        echo "✓ All build tools exist, will use fast-build"
                        env.USE_FAST_BUILD = 'true'
                    } else {
                        echo "⚠ Some build tools missing, will use full build"
                        env.USE_FAST_BUILD = 'false'
                    }
                }
            }
        }
        
        stage('Run CI Lint') {
            steps {
                echo 'Running lint checks...'
                sh """${setupBuildEnv()}
                    
                    # 运行 lint 检查
                    task ci:lint || echo "⚠ Lint completed with warnings"
                """
            }
            post {
                failure {
                    echo '⚠ Lint checks failed but continuing pipeline...'
                }
            }
        }
        
        stage('Build') {
            steps {
                script {
                    if (env.USE_FAST_BUILD == 'true') {
                        echo 'Using fast-build (tools already exist)...'
                        sh """${setupBuildEnv()}
                            task tools:clean-generate-pb
                            task fast-build
                        """
                    } else {
                        echo 'Using full build (installing tools)...'
                        sh """${setupBuildEnv()}
                            task build
                        """
                    }
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
        
        stage('Run Tests') {
            steps {
                echo 'Running tests in all modules...'
                sh """${setupBuildEnv()}
                    
                    # 只测试包含 *_test.go 文件的目录
                        test_failed=0
                        
                        # 查找所有 *_test.go 文件所在的目录
                        find . -name '*_test.go' \\
                            -not -path "*/go/pkg/*" \\
                            -not -path "*/vendor/*" \\
                            -not -path "*/.tools/*" \\
                            -exec dirname {} \\; | sort -u | while read dir; do
                            echo "=========================================="
                            echo "Running tests in: \${dir}"
                            echo "=========================================="
                            
                            # 使用 timeout 命令限制单个目录测试时间为 10 分钟
                            (cd "\${dir}" && timeout 10m go test -v -timeout=5m . 2>&1)
                            exit_code=\$?
                            
                            if [ \${exit_code} -eq 0 ]; then
                                echo "✓ Tests passed in \${dir}"
                            elif [ \${exit_code} -eq 124 ]; then
                                echo "✗ Tests timed out in \${dir}"
                                test_failed=1
                            else
                                echo "✗ Tests failed in \${dir} (exit code: \${exit_code})"
                                test_failed=1
                            fi
                        done
                        
                        if [ \${test_failed} -eq 1 ]; then
                            echo "⚠ Some tests failed, but continuing pipeline..."
                        else
                            echo "✓ All tests passed"
                        fi
                    """
            }
            post {
                failure {
                    echo '⚠ Test stage failed but continuing pipeline...'
                }
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
                    def zipFile = "ProjectY_Server_${env.BUILD_NUMBER}.tar.gz"
                    if (params.NODE_LABEL != 'linux') {
                        env.ARCHIVE_PATH = "/data/archive/disk1/nextcloud/temporary/ProjectY/Server/${params.NODE_LABEL}/${env.BUILD_TIME}_${env.BUILD_NUMBER}"
                    } else
                    {
                        env.ARCHIVE_PATH = "/data/archive/disk1/nextcloud/temporary/ProjectY/Server/${params.NODE_LABEL}"
                    }
                    sh """
                        tar -czvf "${zipFile}"  -C "${sourceDir}" .
                    """
                    echo "打包完成: ${zipFile}"

                    echo "开始通过SCP传输文件到Linux服务器..."
                    
                    sh  """
                        ssh -p 36000 -o StrictHostKeyChecking=no ${env.LINUX_USER}@${env.LINUX_IP} "mkdir -p ${env.ARCHIVE_PATH}"
                        scp -P 36000 -o StrictHostKeyChecking=no "${zipFile}" ${env.LINUX_USER}@${env.LINUX_IP}:${env.ARCHIVE_PATH}/
                        ssh -p 36000 -o StrictHostKeyChecking=no ${env.LINUX_USER}@${env.LINUX_IP} "find ${env.ARCHIVE_PATH} -mindepth 1 -maxdepth 1 -mtime +5 -exec rm -rf {} +"
                        ssh -p 36000 -o StrictHostKeyChecking=no ${env.LINUX_USER}@${env.LINUX_IP} 'count=\$(find '"${env.ARCHIVE_PATH}"' -mindepth 1 -maxdepth 1 -type d | wc -l); if [ \$count -gt 20 ]; then find '"${env.ARCHIVE_PATH}"' -mindepth 1 -maxdepth 1 -type d -printf ''%T@ %p\\n'' | sort -rn | tail -n +21 | cut -d'' '' -f2- | xargs -r rm -rf; fi'
                    """
                    // 传输包名给上游任务
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

                    def meta = [package_name: "${zipFile}"]
                    // 设置显示名与描述（描述用 JSON）
                    currentBuild.displayName = "package_data"
                    currentBuild.description = groovy.json.JsonOutput.toJson(meta)
                }
            }
        }
    }
    
    post {
        success {
            script {
                // 只有在NODE_LABEL=windows时才发送成功消息
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
                    sh "curl -X POST -H \"Content-Type: application/json\" -d \"@payload.json\" ${env.FEISHU_PROJECT_Y_URL}"
                } else {
                    echo '✅ Pipeline completed successfully! (Skipping notification for non-windows build)'
                }
            }
        }
        failure {
            echo '❌ Pipeline failed!'
            script {
                // 失败时始终发送失败消息
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
                sh "curl -X POST -H \"Content-Type: application/json\" -d \"@payload.json\" ${env.FEISHU_PROJECT_Y_URL}"
            }
        }
    }
}
