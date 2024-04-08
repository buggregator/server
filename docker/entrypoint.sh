#!/bin/bash

php app.php configure
php app.php migrate --force

# serve server
./rr serve -c .rr-prod.yaml
