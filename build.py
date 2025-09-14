# bin python3
# python3 build.py
import os
import argparse
import time
import shutil

upx = True

parser = argparse.ArgumentParser(description="of argparse")
parser.add_argument('-name','--name', default='totpl')
parser.add_argument('-path','--path',default="./bin")
parser.add_argument('-port','--port',default="")
parser.add_argument('-os','--os',default="linux")

args = parser.parse_args()
_os = args.os
name = args.name
path = args.path
port = args.port

if port:  # 当 port 不为 None, "", 0, False 等假值时执行
    name = name + port

build_time = time.strftime("%Y-%m-%d")

_os_lowwer = _os.lower()
if _os_lowwer.startswith("lin"):
    print("为 linux 操作系统构建应用")
    os.environ['GOOS']="linux"
    os.environ['GOARCH']="amd64"
elif  _os_lowwer.startswith("win"):
    print("为 windows 操作系统构建应用") 
    name = name + '.exe'
    os.environ['GOOS']="windows"
    os.environ['GOARCH']="amd64"
else:
    print("目前不支持此操作系统：{}, 构建即将终止。".format(_os))
    exit()

print('即将开始 build，参数如下：path= {}'.format(path))
project_root =current_path = os.getcwd()

os.chdir(project_root)

command = '''go build -gcflags=-m -ldflags "-s -w -X 'main.BUILD_TIME={}'" -o {}/{}'''.format(build_time,path,name)

f = os.popen(command)
# f.readlines()
f.close()
del os.environ['GOOS']
del os.environ['GOARCH']
print('build 完毕')

print('开始压缩可执行程序')
if upx:
    zip_file = os.popen('upx -9 {}/{}'.format(path,name))
    zip_file.close()
print('压缩完毕')
