#!/usr/bin/env python3

import sys
import time

print("Printed immediately.")

if len(sys.argv) > 1:
    sleep_time = float(sys.argv[1])
else:
    sleep_time = 2

time.sleep(sleep_time)

print("Printed after " + str(sleep_time))

exit(0)
