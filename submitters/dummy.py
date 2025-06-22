#!/bin/env python3

import sys
import random
import hashlib

for line in sys.stdin:
    # compute some hash to simulate a delay
    for i in range(1000):
        hashlib.sha256(line.encode()).hexdigest()

    status = random.choice(["OK", "ERROR"])
    print(line.strip(), status, "dummy message")
