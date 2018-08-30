# Portal

Portals are a way to ingest content into a CVMFS server.

## Design document

### S3 based

We are basing the portal implementation on the S3 interface, this because S3 is
easily available to anybody.

In particular we will consider in the portal design the implementation of S3
of:

1. AWS
2. minio
3. CEPH (the one use at CERN)

### Bucket semantics

The portal will expect buckets with the name equals to the name of the CVMFS
repository, and an extra bucket with the name `$REPONAME.portal`

If any of those repository is missing, the portal deamon will not consider any
bucket.

As an example, if the portal is managing the repository `foo.cern.ch` and
`bar.fermilab.com` it will expect 4 buckets:

1. foo.cern.ch
2. foo.cern.ch.portal
3. bar.fermilab.com
4. bar.fermilab.com.portal

The `*.portal` bucket will be used to communicate with the external world the
status of the system, we will refer to it as the `status` bucket from now on.

### Portal daemon

The portal daemon will receive as input a TOML document with enumerated the S3
credential along with the domain, we expect an array of table, under the key
`backend`

Example:

``` # foo.cern.ch <- this line is a comment [[backend]] S3_ACCESS_KEY="..."
S3_SECRET_KEY="..." S3_DOMAIN="..."

[[backend]] S3_ACCESS_KEY="..." S3_SECRET_KEY="..." S3_DOMAIN="..." ```

For each backend the daemon will try to connect and will list all the bucket
available.

If it finds a couple of buckets that follow the convention above and if the
daemon is capable of open a transaction in the respective CVMFS repository, it
will start to work on those buckets.

Being carefull with the keys is possible to have a single S3 instance being the
storange for any number of portals

### Daemon

The daemon will be as much parallel as possible.

There will be an indipendent process entity for each repository the deamon work
on.

On top of that there will be an extra one for each couple of bucket tha the
deamon has encounter.

And finally another indipendent unit for each backend defined.

These "indipendent units" will be green-thread or go-routines or even erlang
process, depending on the final implementation of the portal.

What is important is that they are cheap and indipendet from each other.

Now we will explore what each of these indipendet units (process from so on)
do.

#### Backend process.

For each backend defined in the input file, we will spawn a process.

The process will continuosly list the buckets, merge this information with the
repository the daemon is managing, and for each repository managed by the
portal daemon will spawn:

1. A ping process
2. A repo process

~~It will not try to kill the processes when they are not needed anymore since
each one of them will commit suicide as soon as it detect that it should not
exists anymore.~~

We may want to stop a particular portal for a while without changing the
credentials.

Another idea could be to put a file into the status bucket (the `*.portal`
bucket) with a meaning, in this way the operation of the portals could be
stopped manually without touching the configuration file.

The files could be like:

1. RUN
2. STOP

Or maybe, even better, a single file with the action inside. No, a single file
will suffer from eventual consistency issues when we overwrite its content.

Maybe is better to use multiple files and exploit the "last modified" meta-data
key, still possible to face problems with the eventual consistency deal but
very unlikely.

#### Pinging process

The repo processing will simply keep pinging the S3 backend an upload a file
with the timestamp of the last successfull connection into the status bucket,
the `*.portal` one.

#### Repository process

The repository project list all the object into the bucket it is associated
with.

It picks one and check if the file is already been uploaded into the repository
itself, if it is, it pick another one, and so forth in loop.

If the file is not in the repository it start the upload procedure.

After the upload it start again.

### Manages deletions

S3 provides only enventual consistency for deleting operation, this means that
after deleting a file we have no guarantee that a successive list will not find
the file just delete there.

The simplest solution would be to just ignore the problem and in the unlikely
case that the delete is too slow we just upload the same file twice, not an
huge issue.

Another solution is too keep listing the files in the bucket untill we are sure
that, at least in our region, the files is been removed from the index.

Another solution is to use lockfiles, uploading a `$name.tar.hash().STATUS` 
where status would be one of the following:

1. Downloading
2. Ingesting
3. Success
4. Failure
5. Deleted
6. Retry

Those files could be used by the operator to understand what is going on or at
what point of the process we are.

Moreover we could include the timestamp inside those files so to know at what
point of the process each file is, detect inconsistencies, manage retries and
so on.

The status files can be deleted by another process after some time (maybe 24
hours?).

I am quite keen to go for the last option, that is a little more complex but
allow a lot more monitoring in the system, that will be quite essential.
<<<<<<< HEAD


## Details for every process

### Backend process

This process is responsible to keep loading the configuration file and see if
is necessary to spawn other process.

It will keep reloading the configuration file every 30 seconds or when it
receives the SIGHUP signal.

It will start parsing the output of `cvmfs_server list` to understand which
repository are available.

[For each line we split at the first space ' ' and select the first chunk that
should be the name of the repository.]

For every backend the process will try to connect to it using the S3 API.  If
it fails it print an error message.

Once we are connected to a backend we procede to list all the bucket in that
backend.

If we find any bucket that match one of the repository above we check if there
is also the status bucket (`$REPONAME.portal` bucket).

If all those check success we proced to spawn a new `PING` process and a
`Repository` process passing to both the connection configuration and the name
of the repository.

### PING process

The role of this process is to simply give feedback to the operator that the
portal is working correctly.

It simply keep upload the same file `PING` over and over with the content set
to the current timestamp.

It does so every 5 minutes, configurable.

### Backend process

This process is the one that does the real work.

#### Understand if it should run

To understand if the process should run we start by listing the status bucket.

The process start working if any of the following is true.

1. The bucket is empty
2. There is only the PING file
3. A RUN file is present and a STOP file is not present
4. Both a RUN file and a STOP file is prensent, but the last-modified file of
   the RUN file is successive to the one of the STOP file.
5. There is not a STOP file.

Condition 1) and 2) collapse into 5)

In all other case the process should not do any work, it will start a timeout
and repeat the check after 5 minutes.

#### Running

If we decide that the process should run it start by listing the content of the
main bucket.

It will sort the files by the last-modified field and start analyze each file.

It will start by computing the hash of the stat of the file.

With the name of the file and its metadata HASH will start to look for the
status files.

It will continue the procedure if and only if:

1. No status file are present
2. Both `Failure` and `Retry` file are present with the `Retry` file being
   newer tham the `Failure` one

If the process decides to continue it will start downloading the file.

#### DOWNLOADING

In this phase we are moving the file from the S3 backend into our local
storage.

We start by creating a temporary file.

We then upload the `.DOWNLOADING` file into the status bucket and we try to
download the file writing it into the temporary file just created.

We re-try the download 3 times (configurable) in case of error. 

If still we are unable to download the file we write the `.FAILURE` file
logging the error, we delete the `.DOWNLOADING` file and we move on the next
file.

For each failure we log it into STDERR.

#### INGESTING

Once the file is in the local storange we can proced to ingest it into CVMFS.

We start by uploading the `.INGESTING` file into the status bucket.

Again we try to ingest the file a configurable number of time.  (Can I
understand what error the `cvmfs_server` return without parsing the output?)

If we were unsucessfull in ingesting the file we upload the `.FAILURE` file
into the status bucket,then we delete the `.INGESTING` file and finally we
clean up the temporary file and we move to the next file.

If we are successful in ingesting the file we upload the `.SUCCESS`.

#### DELETING

Once we successfully upload a file we proces to delete the file from S3.

We start by uploading the `.DELETE` status file.

Then we issue the delete command.

#### Final state

If everything went correctly we should have the file ingested into CVMFS, the
`.DOWNLOADING`, `.INGESTING`, `.DELETED` and `.SUCESS` files into the status
bucket and not anymore the original file into the bucket.

If there was any error we should have the `.FAILURE` status files with the
error logged in, along with the status file of the last attempt operation.

In the case it was a retry attempt, the previous status files would have been
overwritten by the next one.

After all the files from the listing have been analyzed, we issue another list
operation and if there are new files to work on we start to ingest them,
otherwise we sleep for 10 minutes.


### Garbage Collection

For every sucessfull upload we generate 4 status files, after a while all this
status files could really get into the way of the operator.

We can spawn a garbage collector process that will clean up all those files.

A reasonable default would be to left untouched all the files relative to a
failed attempt (no matter how old) and delete all the file returned by a
sucessfull attempt after 24 hours.

EOF
