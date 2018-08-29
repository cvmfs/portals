# Portal

Portals are a way to ingest content into a CVMFS server.

## Design document

### S3 based

We are basing the portal implementation on the S3 interface, this because S3 is 
easily available to anybody.

In particular we will consider in the portal design the implementation of S3 of:

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

```
# foo.cern.ch <- this line is a comment
[[backend]]
S3_ACCESS_KEY="..."
S3_SECRET_KEY="..."
S3_DOMAIN="..."

[[backend]]
S3_ACCESS_KEY="..."
S3_SECRET_KEY="..."
S3_DOMAIN="..."
```

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

It will not try to kill the processes when they are not needed anymore since
each one of them will commit suicide as soon as it detect that it should not
exists anymore.

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

Those files could be used by the operator to understand what is going on or at
what point of the process we are.

Moreover we could include the timestamp inside those files so to know at what
point of the process each file is, detect inconsistencies, manage retries and
so on.

The status files can be deleted by another process after some time (maybe 24
hours?).

I am quite keen to go for the last option, that is a little more complex but
allow a lot more monitoring in the system, that will be quite essential.
