#!/usr/bin/env python3
import hashlib
import os
import random
import string

import requests

print(
    "Hello! I am a little sploit. I could be written on any language, but "
    "my author loves Python. Look at my source - it is really simple. "
    "I should steal flags and print them on stdout or stderr. "
)

# The farm host ip and port
cache_url = os.environ.get("CACHE_URL")

# The host to attack is passed as the first argument
target = os.environ.get("TARGET")

# I want to get the flagids but cache them!
flagids_url = f"{cache_url}https://example.com"
res = requests.get(flagids_url)
print(f"Got flagids from {flagids_url}: {res.text}")


# simulate some computation
if random.choice([True, False]):
    for i in range(1000000):
        txt = [random.choice(string.ascii_uppercase + string.digits) for _ in range(31)]
        txt = "".join(txt)
        hashlib.sha256(txt.encode())

print(f"I need to attack a team with host: {target}")
print(f"I can query the farm at: {cache_url}")
print("Here are some random flags for you:")


def spam_flag():
    arr = [random.choice(string.ascii_uppercase + string.digits) for _ in range(31)]
    flag = "".join(arr) + "="
    print(flag, flush=True)


spam_flag()
