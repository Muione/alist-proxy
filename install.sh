#!/bin/bash

REPO="Muione/alist-proxy"

INSTALL_PATH="/opt/alist-proxy"
ACTION="install"

# 处理命令行参数
while getopts "d:iur" opt; do
  case $opt in
    d)
      INSTALL_PATH="$OPTARG"
      ;;
    i)
      ACTION="install"
      ;;
    u)
      ACTION="update"
      ;;
    r)
      ACTION="uninstall"
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 1
      ;;
  esac
done

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

function download_alist_proxy() {
    curl -LO https://github.com/${REPO}/releases/download/v${TAG}/alist-proxy_${TAG}_${OS_TYPE}_${ARCH}.tar.gz
    tar -zxvf alist-proxy_${TAG}_${OS_TYPE}_${ARCH}.tar.gz
    rm alist-proxy_${TAG}_${OS_TYPE}_${ARCH}.tar.gz
    chmod +x alist-proxy
}

if [ "$ACTION" == "install" ]; then
    mkdir -p ${INSTALL_PATH}
    cd ${INSTALL_PATH}
    download_alist_proxy
    
    # 下载service模板：
    curl -LO https://raw.githubusercontent.com/${REPO}/main/alist-proxy.service
    # 替换service模板中的路径：
    sed -i "s|/opt/alist-proxy|${INSTALL_PATH}|g" alist-proxy.service
    # 安装service：
    sudo cp alist-proxy.service /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable alist-proxy
    # 预运行生成默认配置文件
    ${INSTALL_PATH}/alist-proxy
    # 输出安装完成信息：
    echo "alist-proxy has been installed successfully."
    echo "Please edit ${INSTALL_PATH}/config.yaml and then run :"
    echo "sudo systemctl start alist-proxy"
elif [ "$ACTION" == "uninstall" ]; then
    # 卸载service：
    sudo systemctl disable alist-proxy
    sudo systemctl stop alist-proxy
    sudo rm /etc/systemd/system/alist-proxy.service
    sudo systemctl daemon-reload
    
    echo "alist-proxy has been uninstalled successfully."
    echo "please remove the alist-proxy directory manually"

elif [ "$ACTION" == "update" ]; then
    sudo systemctl stop alist-proxy
    # 下载最新版本：
    cd ${INSTALL_PATH}
    mv alist-proxy alist-proxy.bak
    download_alist_proxy
    rm alist-proxy.bak
    sudo systemctl restart alist-proxy
    echo "alist-proxy has been updated successfully."
else
    echo "Invalid action.."
fi


