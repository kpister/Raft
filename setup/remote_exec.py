"""Setup
Usage:
    remote_exec.py <ip_file> <pem_file> [options]

Options:
    -h, --help          Print help message and exit
    -s, --sleep INT     Sleep time [default: 2]
"""

from docopt import docopt
from os import path
import json
import paramiko
import time
import warnings
warnings.filterwarnings("ignore")

RAFT_PATH = "/home/ec2-user/go-workplace/src/github.com/kpister/raft"

def connect_and_exec(ip, cmd, pem, sleep_time):
    # connect to server
    print(f'Connecting to {ip}')
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        client.connect(hostname=ip, username="ec2-user", pkey=pem, timeout = 2)
    except:
        return

    print("Executing command :: " + cmd)
    client.exec_command(cmd)

    time.sleep(sleep_time)
    
    client.close()

if __name__ == '__main__':
    args = docopt(__doc__)

    sleep_time = int(args['--sleep'])

    if not path.exists(args['<ip_file>']):
        raise Exception("IP file does not exist")

    if not path.exists(args['<pem_file>']):
        raise Exception(".pem file does not exist")

    ips = []

    for line in open(args['<ip_file>']):
        ips.append(line.strip())

    pem = paramiko.RSAKey.from_private_key_file(args['<pem_file>'])
    
    while True:
        cmd = input("$ ")
        if cmd == "quit":
            break
        for ip in ips:
            connect_and_exec(ip, cmd, pem, sleep_time)
