#!/bin/bash

# Ensure required directories exist
mkdir -p /home/student/workspace
mkdir -p /home/student/share

# Ensure correct ownership
chown -R student:student /home/student/workspace
chown -R student:student /home/student/share

echo 'echo "Welcome to the linuxus service shell."' >> /home/student/.bashrc
echo 'echo "Personal folder: ~/workspace"' >> /home/student/.bashrc
echo 'echo "Shared folder  : ~/share"' >> /home/student/.bashrc
echo 'if [ -n "$STUDENT_NAME" ]; then echo "Student: $STUDENT_NAME"; fi' >> /home/student/.bashrc

# Start ttyd and open bash in the student's home directory
exec ttyd \
  --writable \
  --port 7681 \
  sudo -u student -H bash -lc "cd /home/student && exec bash"