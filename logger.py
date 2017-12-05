#!/usr/bin/python
# -*- coding: UTF-8 -*-

__version__ = '1.0.0.0'

import logging

# 指定打印日志的级别
logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s %(filename)s[line:%(lineno)d] %(levelname)s %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S',
        filename='mssh.log',
        filemode='a'
)

# 输出到控制台的日志与级别
console = logging.StreamHandler()
console.setLevel(logging.ERROR)
formatter = logging.Formatter('%(asctime)s %(filename)s[line:%(lineno)d] %(levelname)s %(message)s')
console.setFormatter(formatter)

LOG = logging.getLogger('')
LOG.addHandler(console)

if __name__ == '__main__':
    LOG.error('e')
