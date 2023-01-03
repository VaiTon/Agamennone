#!/bin/env python3
import json
flags = json.loads(input())

# Add your code here

# Example:
import random
for flag in flags:
    if random.randint(0, 1) == 0:
        flag["status"] = "REJECTED"
    else:
        flag["status"] = "ACCEPTED"

# End of your code

print(json.dumps(flags), flush=True)

