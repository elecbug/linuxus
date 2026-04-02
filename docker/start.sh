#!/bin/bash

# Ensure correct ownership for the home directory
chown -R student:student /home/student

# Start ttyd and launch bash as the student user
exec ttyd \
  --writable \
  --port 7681 \
  sudo -u student -H bash -lc "cd /home/student && exec bash"