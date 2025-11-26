#!/usr/bin/env python3
"""
Publish script for gh-pages deployment using Docker.
This version is designed to run inside a Docker container.
"""
from __future__ import division, print_function, absolute_import, unicode_literals
import argparse
import os
import tempfile
import shutil
import sys
import subprocess

# constants
PY3 = sys.version_info[0] == 3
PY2 = sys.version_info[0] == 2

# global variables
verbose = 0
quiet = False
public_dir = "public"
branch = "gh-pages"
project_dir = os.path.dirname(os.path.abspath(__file__))
commit_message = "Publish to gh-pages by publish script"

# functions

def debug(s):
    if verbose >= 1 and not quiet:
        print(s)

def info(s):
    if not quiet:
        print(s)

def abort(s):
    print(colors.red + s + colors.reset, file=sys.stderr)
    sys.exit(1)

def run_command(cmd):
    p = subprocess.Popen(
        cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout_data, stderr_data = p.communicate()
    if p.returncode != 0:
        abort("failed to execute command: {}\nstderr: {}".format(cmd, stderr_data))
    if PY3:
        stdout_data = stdout_data.decode('utf-8')
    return stdout_data

# classes for pseudo-namespaces.

class colors:
    bold = '\033[1m'
    underlined = '\033[4m'
    red = '\033[31m'
    green = '\033[32m'
    yellow = '\033[33m'
    blue = '\033[34m'
    reset = '\033[0m'

# main

def main():
    os.chdir(project_dir)

    parser = argparse.ArgumentParser(
        description="gh-pages-publish is a CLI tool to publish Github Pages.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Adapted for Remembrances MCP
"""
    )

    parser.add_argument("-v", "--verbose", action="count",
                        help="Increase output verbosity", default=0)
    parser.add_argument("-q", "--quiet", dest="quiet",
                        action="store_true", help="Do not output any info.")
    args = parser.parse_args()

    global verbose
    verbose = args.verbose
    global quiet
    quiet = args.quiet

    # Check if public directory exists
    if not os.path.exists(public_dir):
        abort(f"Public directory '{public_dir}' not found. Run 'make build' first.")

    info("Publishing '{}' directory to {} branch...".format(public_dir, branch))

    # create temporary directory
    tempdir = tempfile.mkdtemp(prefix="gh-pages-publish_")
    debug("Created temporary directory: " + tempdir)

    remote = run_command("git config --get remote.origin.url").strip()

    cmd = "cd {} && git clone --quiet {} repo && cd repo && git checkout {} && rm -rf ./*".format(
        tempdir, remote, branch)
    debug("cmd: " + cmd)
    run_command(cmd)

    cmd = "rsync -a {}/ {}/repo/".format(public_dir, tempdir)
    debug("cmd: " + cmd)
    run_command(cmd)

    cmd = "cd {}/repo/ && git add -A && git commit -am '{}' && git push --force origin {} --quiet".format(
        tempdir, commit_message, branch)
    debug("cmd: " + cmd)
    run_command(cmd)

    # remove temporary directory
    shutil.rmtree(tempdir)
    debug("Removed temporary directory: " + tempdir)

    info("Done.")

if __name__ == '__main__':
    main()
