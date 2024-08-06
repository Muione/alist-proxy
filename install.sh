REPO="Muione/alist-proxy"

INSTALL_PATH="/opt/alist-proxy"

# 检查是否传入了 -d 参数
if [ "$1" == "-d" ]; then
    # 检查 $2 是否为空
    if [ -z "$2" ]; then
        echo "请指定安装路径"
        exit 1
    fi
    # 如果$2是相对路径，还需要改成绝对路径
    if [ "${2:0:1}" != "/" ]; then
        INSTALL_PATH=$(pwd)/$2
    else
        INSTALL_PATH=$2
    fi
fi

OS_TYPE="linux"
ARCH="amd64"
TAG="1.0.0"

# 获取当前系统类型：
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS_TYPE="linux";
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS_TYPE="darwin";
elif [[ "$OSTYPE" == "win32" ]]; then
    OS_TYPE="windows";
else
    echo "Unsupported OS type: $OSTYPE";
    exit 1;
fi

# 获取当前系统架构(386 amd64 arm64)：
if [[ "$(uname -m)" == "x86_64" ]]; then
    ARCH="amd64";
elif [[ "$(uname -m)" == "i386" ]]; then
    ARCH="386";
elif [[ "$(uname -m)" == "armv6l" ]]; then
    ARCH="armv6";
elif [[ "$(uname -m)" == "aarch64" ]]; then
    ARCH="arm64";
else
    echo "Unsupported architecture: $(uname -m)";
    exit 1;
fi

# 获取最新版本号：
TAG=$(curl -s https://api.github.com/repos/${REPO}/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')
TAG=${TAG#v} # 去掉v前缀

# 创建 alist-proxy 文件夹
mkdir -p ${INSTALL_PATH}

# 进入 alist-proxy 文件夹
cd ${INSTALL_PATH}

# 下载并安装：
# 文件名：alist-proxy_1.0.2_darwin_amd64.tar.gz
curl -LO https://github.com/${REPO}/releases/download/v${TAG}/alist-proxy_${TAG}_${OS_TYPE}_${ARCH}.tar.gz
tar -zxvf alist-proxy_${TAG}_${OS_TYPE}_${ARCH}.tar.gz
rm alist-proxy_${TAG}_${OS_TYPE}_${ARCH}.tar.gz

chmod +x alist-proxy

# 下载service模板：
curl -LO https://raw.githubusercontent.com/${REPO}/main/alist-proxy.service

# 替换service模板中的路径：
sed -i "s|/opt/alist-proxy|${INSTALL_PATH}|g" alist-proxy.service

# 安装service：
sudo cp alist-proxy.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable alist-proxy
# sudo systemctl start alist-proxy

# 预运行生成默认配置文件
${INSTALL_PATH}/alist-proxy


# 输出安装完成信息：
echo "alist-proxy has been installed successfully."
echo "Please edit ${INSTALL_PATH}/config.yaml and then run :"
echo "sudo systemctl start alist-proxy"


