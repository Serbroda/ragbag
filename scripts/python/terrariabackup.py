#!/usr/bin/env python
# -*- coding: UTF-8 -*-

import pickle
import time
import zipfile
import os
import hashlib
import getopt
import sys

# https://stackoverflow.com/questions/27913261/python-storing-data
# https://www.geeksforgeeks.org/working-zip-files-python/
# https://stackoverflow.com/questions/3431825/generating-an-md5-checksum-of-a-file

ARGS_KEY_CWD = 'cwd'
ARGS_KEY_INPUT = 'input'
ARGS_KEY_OUTPUT = 'output'
ARGS_KEY_DATA_FILE = 'data'
ARGS_KEY_TIMESTAMP = 'timestamp'
ARGS_KEY_FORCE = 'force'

DEFAULT_DATA_FILE = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), 'terrariabackup.data')
DATA_KEY_LAST_FHASH = 'last_fhash'
DATA_KEY_LAST_BACKUP_FILE = 'last_backup_file'
DATA_KEY_LAST_FILE_COUNT = 'last_file_count'

args = {}


def get_args():
    args[ARGS_KEY_DATA_FILE] = DEFAULT_DATA_FILE
    args[ARGS_KEY_TIMESTAMP] = 0
    args[ARGS_KEY_FORCE] = 0

    # Get full command-line arguments
    full_cmd_arguments = sys.argv

    # Keep all but the first
    argument_list = full_cmd_arguments[1:]

    short_options = "hi:o:c:d:tf"
    long_options = ["help", ARGS_KEY_INPUT + "=", ARGS_KEY_OUTPUT +
                    "=", ARGS_KEY_CWD + "=", ARGS_KEY_DATA_FILE + "=", ARGS_KEY_TIMESTAMP, ARGS_KEY_FORCE]

    arguments, values = getopt.getopt(
        argument_list, short_options, long_options)

    for current_argument, current_value in arguments:
        if current_argument in ("-h", "--help"):
            print("Command line argument list:")
            print("")
            print("------------------------------------------")
            print("long argument   short argument  with value")
            print("------------------------------------------")
            print("--help           -h              no")
            print("--input          -i              yes")
            print("--output         -o              yes")
            print("--timestamp      -t              no")
            print("--force          -f              no")
            print("--cwd            -c              yes")
            print("------------------------------------------")
            exit(2)
        elif current_argument in ("-i", "--" + ARGS_KEY_INPUT):
            args[ARGS_KEY_INPUT] = current_value.split(',')
        elif current_argument in ("-o", "--" + ARGS_KEY_OUTPUT):
            args[ARGS_KEY_OUTPUT] = current_value
        elif current_argument in ("-d", "--" + ARGS_KEY_DATA_FILE):
            args[ARGS_KEY_DATA_FILE] = current_value
        elif current_argument in ("-t", "--" + ARGS_KEY_TIMESTAMP):
            args[ARGS_KEY_TIMESTAMP] = 1
        elif current_argument in ("-f", "--" + ARGS_KEY_FORCE):
            args[ARGS_KEY_FORCE] = 1
        elif current_argument in ("-c", "--" + ARGS_KEY_CWD):
            try:
                os.chdir(current_value)
            except OSError:
                print("Can't change the Current Working Directory to {}".format(
                    current_value))

    if ARGS_KEY_INPUT not in args:
        print('Input must be specified')
        exit(3)
    elif ARGS_KEY_OUTPUT not in args:
        print('Output must be specified')
        exit(3)

    return args


def save_data(dataset):
    datafile = args[ARGS_KEY_DATA_FILE]

    fw = open(datafile, 'wb')
    pickle.dump(dataset, open(datafile, "wb"))
    fw.close()


def load_data():
    datafile = args[ARGS_KEY_DATA_FILE]

    if not os.path.exists(datafile):
        save_data({
            DATA_KEY_LAST_FHASH: -1,
            DATA_KEY_LAST_BACKUP_FILE: '',
            DATA_KEY_LAST_FILE_COUNT: 0
        })
    return pickle.load(open(datafile, "rb"))


def get_all_file_paths(paths):
    file_paths = []

    for p in paths:
        if os.path.isfile(p):
            file_paths.append(p)
        elif os.path.isdir(p):
            for root, directories, files in os.walk(p):
                for filename in files:
                    filepath = os.path.join(root, filename)
                    file_paths.append(filepath)
        else:
            if os.path.exists(p):
                print("File {} is a special file (socket, FIFO, device file)".format(p))

    return file_paths


def archive_files(paths, zip_name):
    files = get_all_file_paths(paths)

    with zipfile.ZipFile(zip_name, 'w', zipfile.ZIP_DEFLATED) as zip_file:
        for f in files:
            zip_file.write(f)


def calculate_md5(paths):
    files = get_all_file_paths(paths)
    hash_md5 = hashlib.md5()

    for p in files:
        if os.path.isfile(p):
            with open(p, "rb") as f:
                for chunk in iter(lambda: f.read(4096), b""):
                    hash_md5.update(chunk)

    return hash_md5.hexdigest()


def create_filename_with_timestamp(filename):
    timestr = time.strftime("%Y%m%d-%H%M%S")
    dir = os.path.dirname(filename)
    base = os.path.basename(filename)
    parts = os.path.splitext(base)
    filenamebase = parts[0]
    ext = ''
    if len(parts) > 1:
        ext = parts[1]
    targetfilename = filenamebase + '-' + timestr + ext
    return os.path.join(dir, targetfilename)


def main():
    args = get_args()

    print('')
    print('----------------------------------------------')
    print('Starting programm with args: {}'.format(args))

    data = load_data()

    print('Last data: {}'.format(data))
    print('----------------------------------------------')
    print('')

    all_files = get_all_file_paths(args[ARGS_KEY_INPUT])
    files_hash = calculate_md5(all_files)

    if files_hash != data[DATA_KEY_LAST_FHASH] or \
            not os.path.exists(data[DATA_KEY_LAST_BACKUP_FILE]) \
            or data[DATA_KEY_LAST_FILE_COUNT] != len(all_files) \
            or args[ARGS_KEY_FORCE] == 1:

        target_file = args[ARGS_KEY_OUTPUT]
        if args[ARGS_KEY_TIMESTAMP] == 1:
            target_file = create_filename_with_timestamp(target_file)

        print('Archiving paths {} to {}'.format(
            args[ARGS_KEY_INPUT], target_file))
        archive_files(all_files, target_file)
        data[DATA_KEY_LAST_FHASH] = files_hash
        data[DATA_KEY_LAST_BACKUP_FILE] = target_file
        data[DATA_KEY_LAST_FILE_COUNT] = len(all_files)
        save_data(data)
    else:
        print('Nothing changed, no need to backup')


if __name__ == "__main__":
    main()
