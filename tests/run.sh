#!/bin/sh
# This doesn't work so well, bash is perhaps not the best for writing 
# stuff like this
set -e
# Test write permissions
touch first-failed server-failed
(cd ..; make)
set +e
rm -f test.db first-failed server-failed
sqlite3 test.db < init.sql
../terraform-state-server sqlite://test.db || touch server-failed &
sleep 2
if [ -f "server-failed" ]; then
	echo "Server run failed"
	exit 1
fi
trap "pkill -f 'terraform-state-server sqlite://test.db'; sleep 0.1; rm -f server-failed" EXIT INT
FAILED=0
terraform init > /dev/null || {
	echo "Terraform init failed"
	kill %1
	exit 1
}
echo "Starting first apply. This should succeed"
FIRST=$(terraform apply -auto-approve > /dev/null 2>&1 || touch first-failed) &
sleep 1
terraform apply -auto-approve > /dev/null 2>&1
if [ "$?" != "1" ]; then
	echo "Apply did not fail? Something is wrong with the locking"
	FAILED=1
fi
echo "Waiting for other terraform to finish"
sleep 5
if [ -f "first-failed" ]; then
	echo "First terraform run did not succeed, something is wrong"
	FAILED=1
fi
if [ "$FAILED" = "0" ]; then
	RES=PASS
else
	RES=FAIL
fi
echo "Test result: $RES"
rm -f test.db first-failed server-failed
exit $FAILED
