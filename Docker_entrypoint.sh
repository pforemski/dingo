#!/bin/sh

cmdArgs="$*"
if [ -n "$cmdArgs" ]; then
  /opt/dingo $cmdArgs
  exit 0
fi

Args=${Args:--gdns:auto -bind=0.0.0.0}

cat > /opt/supervisord.conf <<EOF
[supervisord]
nodaemon=true

[program:dingo]
command=/opt/dingo ${Args}
autorestart=true
redirect_stderr=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0

EOF

/usr/bin/supervisord -c /opt/supervisord.conf
