#!/usr/bin/env python
# -*- coding: utf-8 -*-
__version__ = '1.0.0.0'

import readline
import os
import sys
import traceback
import pymongo
import paramiko
import scp
import threading
from eventlet import greenpool
import atexit
from termcolor import cprint

from logger import LOG

LOG.info('-------------------mission start-------------------------')

print sys.argv

def get_conf():
    confs = {}
    with open('mssh.conf', 'r') as fp:
        lines = fp.readlines()
        for line in lines:
            if line and line[0] != '#':
                temp = line.replace('\n', '').split(' ')
                conf = {
                    'host': temp[0],
                    'user': temp[1],
                    'passwd': temp[2]
                }
                confs[temp[0]] = conf
    return confs

def get_session(ip, username, passwd):
    LOG.info('get session: %s, %s, %s' % (ip, username, passwd))
    try:
        session = paramiko.SSHClient()
        session.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        session.connect(ip, 22, username, passwd, timeout=5)
        #session.close()
        cprint('get session success: %s\n' % (ip, ), 'green')
        LOG.info('get session success: %s\n' % (ip, ))
        return session
    except Exception as ee:
        cprint('get session failed: %s\n' % (ip, ), 'red')
        LOG.error('get session failed: %s\n' % (ip, ))

def upload(ip, ssh, filepath, remotepath='.'):
    LOG.info('upload %s to %s:%s' % (filepath, ip, remotepath))
    try:
        stdin, stdout, stderr = ssh.exec_command('cd ~&pwd')
        out = stdout.readlines()
        #print out
        home = out[0].replace('\n', '')
        #print home
        #sftp = ssh.open_sftp()
        sftp = scp.SCPClient(ssh.get_transport())
        if remotepath == '.':
            remotepath = home + '/' + filepath.split('/')[-1]
        localpath = filepath
        sftp.put(localpath, remotepath)
        cprint('upload %s to %s:%s success' % (localpath, ip, remotepath), 'green')
        LOG.info('upload %s to %s:%s success' % (localpath, ip, remotepath))
    except Exception as ee:
        cprint('upload to %s failed: %s' % (ip, traceback.format_exc()), 'red')
        LOG.error('upload %s to %s failed: %s' % (filepath, ip, traceback.format_exc()))

def download(ip, ssh, filepath):
    LOG.info('download from %s:%s' % (ip, filepath))
    try:
        #print 'download %s from %s' % (filepath, ip)
        #sftp = ssh.open_sftp()
        sftp = scp.SCPClient(ssh.get_transport())
        remotepath = filepath
        paths = ['.', 'download', ip]
        paths.extend(filepath.split('/')[:-1])
        localdir = '/'.join(paths)
        #print localdir
        try:
            os.makedirs(localdir)
        except Exception as ee:
            pass
        paths.append(filepath.split('/')[-1])
        localpath = '/'.join(paths)
        #print localpath
        sftp.get(remotepath, localpath)
        cprint('download from %s:%s to %s success' % (ip, remotepath, localpath), 'green')
        LOG.info('download from %s:%s to %s success' % (ip, remotepath, localpath))
    except Exception as ee:
        cprint('download from %s failed: %s' % (ip, traceback.format_exc()), 'red')
        LOG.error('download from %s failed: %s' % (ip, traceback.format_exc()))


def sshc(host, session, cmds):
    LOG.info('%s exec %s' % (host, cmds))
    try:
        for m in cmds:
            stdin, stdout, stderr = session.exec_command(m)
            #stdin.write("Y")   #简单交互，输入 ‘Y’ 
            out = stdout.readlines()
            #屏幕输出
            for o in out:
                print o,
            for o in stderr.readlines():
                print o,
        cprint('%s exec success' % host, 'green')
        LOG.info('%s exec success' % host)
    except Exception as ee:
        cprint('%s exec error' % host, 'red')
        LOG.error('%s exec error' % host)

def sshs(host, ssh, cmds):
    """
    python远程执行脚本
    """
    try:
        session = ssh.invoke_shell()
        session.send('\n'.join(cmds))
        print session.recv().decode('utf-8')
        cprint('%s success' % host, 'green')
    except Exception as ee:
        cprint('%s error: %s' % (host, ee), 'red')

def sshi(host, session):
    """
    初始化各IP对应的平台
    """
    try:
        stdin, stdout, stderr = session.exec_command('uname -m')
        hardware = stdout.readlines()[0].replace('\n', '')
        confs[host]['hardware'] = hardware
        cprint('%s success' % host, 'green')
    except Exception as ee:
        cprint('%s error: %s' % (host, ee), 'red')

def init_platform():
    for host, session in sessions.iteritems():
        pool.spawn(sshi, host, session)
    pool.waitall()


pool = greenpool.GreenPool(100)

confs = get_conf()
sessions = {}
for k, conf in confs.iteritems():
    session = get_session(conf['host'], conf['user'], conf['passwd'])
    if session:
        sessions[k] = session

init_platform()
print confs

def download_all(filepath):
    for host, session in sessions.iteritems():
        pool.spawn(download, host, session, filepath)

def upload_all(filepath, remotepath='.'):
    for host, session in sessions.iteritems():
        pool.spawn(upload, host, session, filepath, remotepath)

def upload_install():
    LOG.info('upload install')
    install_packs = os.listdir('install')
    for host, session in sessions.iteritems():
        hardware = confs[host].get('hardware')
        filepath = None
        filename = None
        if hardware:
            if 'arm' in hardware:
                for pack in install_packs:
                    if 'rainbow' in pack:
                        filepath = 'install/%s' % pack
                        filename = pack
            else:
                for pack in install_packs:
                    if 'lightning' in pack:
                        filepath = 'install/%s' % pack
                        filename = pack
        if filepath:
            pool.spawn(upload, host, session, filepath, '/opt/%s' % filename)
            confs[host]['filename'] = filename

def upload_backup():
    for host, session in sessions.iteritems():
        filepath = '/'.join(['.', 'download', host, 'opt', 'data', 'backup.tar.gz'])
        pool.spawn(upload, host, session, filepath, '/opt/data/backup.tar.gz')

def exec_all(args):
    cmds = args
    if isinstance(args, str):
        cmds = [args]
    for host, session in sessions.iteritems():
        pool.spawn(sshc, host, session, cmds)

def exec_script(args):
    cmds = args
    if isinstance(args, str):
        cmds = [args]
    for host, session in sessions.iteritems():
        pool.spawn(sshs, host, session, cmds)

def exec_install():
    for host, session in sessions.iteritems():
        hardware = confs[host]['hardware']
        filename = confs[host].get('filename')
        if not hardware:
            continue
        cmds = []
        if 'arm' in hardware:
            cmds.append('tar -zxvf /opt/%s -C /opt/' % filename)
            cmds.append('/opt/au/installer/install')
        else:
            cmds.append('sh /opt/%s -y' % filename)
        pool.spawn(sshc, host, session, cmds)

def done_func(cmds):
    exec_script(cmds)

def input_cmds():
    cmds = []
    while True:
        cmd = raw_input('>')
        func, args = parse(cmd)
        if func == done_func:
            print cmds
            done_func(cmds)
            return
        elif func:
            func(*args)
        else:
            cmds.append(args)

def clear():
    os.system('clear')

cmd_map = {
    'q': sys.exit,
    'put_backup': upload_backup,
    'put_install': upload_install,
    'exec_install': exec_install,
    'put': upload_all,
    'get': download_all,
    'do': input_cmds,
    'done': done_func,
    'clear': clear
}


def parse(cmd):
    args = cmd.split(' ')
    args = filter(lambda x: x != '', args)
    if not args:
        return None, cmd
    c = cmd_map.get(args[0])
    if c:
        # 本地命令
        return c, tuple(args[1:])
    else:
        # 远程命令
        return None, cmd

def exit_func():
    print '关闭会话'
    pool.waitall()
    for host, session in sessions.iteritems():
        session.close()
    print '退出'
    LOG.info('-------------------mission end-------------------------')

atexit.register(exit_func)

while True:
    try:
        cmd = raw_input('[mssh ~ ]# ')
        if cmd.startswith('#'):
            # 过滤注释
            continue
        func, args = parse(cmd)
        #print func, args
        if func:
            func(*args)
        else:
            exec_all(args)
        pool.waitall()
    except Exception as ee:
        print ee
        sys.exit()

