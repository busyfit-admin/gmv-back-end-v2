#!/usr/bin/python3
import os
import random
import string
import time
from behave import *


@given("sleep {n} seconds")
def step_sleep_seconds(context, n):
    time.sleep(int(n))


@when("sleep {n} seconds")
def step_sleep_seconds(context, n):
    time.sleep(int(n))


def read_file_path(file_path):
    return read_file_contents(get_absolute_filepath(file_path))


def get_absolute_filepath(file_path):
    return os.path.join(os.path.dirname(os.path.abspath(__file__)), os.pardir, file_path)


def read_file_contents(file_path):
    with open(file_path, 'r') as file:
        contents = file.read()
        file.close()
    return contents


def id_generator(size=6, chars=string.ascii_uppercase + string.digits):
    return ''.join(random.choice(chars) for _ in range(size))
