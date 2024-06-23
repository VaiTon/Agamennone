#!/bin/env python3

import sys, random, time, hashlib

for line in sys.stdin:
    status = random.choice(["OK", "ERROR"])
    # compute some hash to simulate a delay
    for i in range(1000):
        hashlib.sha256(line.encode()).hexdigest()

    print(line.strip(), status, "dummy message")