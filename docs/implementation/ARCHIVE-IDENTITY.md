# Read-only archive identity check

Run on a quiescent, owner-controlled local archive after the approved self-hosted
macOS build has finished:

```sh
/usr/bin/python3 -I scripts/check-ios-archive.py \
  --archive /path/to/candidate.xcarchive --version 1.0.0 --build 7
```

The numbers are illustrative, not a selected or reserved release. Supply the
reviewed expected version/build; obtain unused-number evidence separately from
App Store Connect. This command performs no build, signing, Keychain access,
network call, installation, mutation or upload.

Exit zero means ONLY that the inspected metadata matches `run.sire.prosody`, the
caller-selected version/build, a single app selected by canonical archive path,
and the supported modern thin ARM64/iOS stamps. XML/binary plist input is bounded
and duplicate keys are rejected. Binary object offsets/extents must be unique
and nonoverlapping; at most 50,000 objects and 100,000 aggregate references are
accepted, with acyclic graph depth at most 64. These checks precede allocation
by the standard plist decoder; legitimate repeated object-index references remain
supported. Files and archive descendant directories are
opened without following symlinks; open descriptors anchor the inspected tree.
Mutation metadata is checked again before returning. Those observations do not
freeze the path permanently, establish an artifact digest or create an upload
capability. Do not race this checker with a producer or substitute another artifact
later. Network/FUSE I/O and hostile concurrent writers are not qualified here;
the CLI alarm does not guarantee interruption of uninterruptible kernel I/O.

This deliberately narrow profile rejects universal/fat, ARM64e, simulator and
legacy-platform-stamp binaries until separately supported. The Mach-O check
inspects header/load-command bounds and the platform stamp only. Synthetic fixtures
with matching stamps are not runnable applications: the checker does not validate
code semantics, full Mach-O loadability, signatures, entitlements, provisioning,
Apple upload/SDK rules, source provenance, runtime/backend behavior or acceptance.
It neither generates nor approves an archive. A native signature/profile audit,
reviewed candidate binding and Apple validation still precede an authorized upload.

The CLI always prints its signature/provisioning/provenance/upload limitations
alongside identity success. Fixed failure messages avoid rendering private
artifact paths or plist values. Helper success is not TestFlight readiness and
must not be used to bypass any existing gate.

Format references: Apple's [Mach-O definitions](https://github.com/apple-oss-distributions/xnu/blob/main/EXTERNAL_HEADERS/mach-o/loader.h)
and [information property lists](https://developer.apple.com/library/archive/documentation/General/Reference/InfoPlistKeyReference/Articles/AboutInformationPropertyListFiles.html).
The constants were also inspected in the installed Xcode iPhoneOS SDK headers;
this source check is not a current native archive qualification receipt.
