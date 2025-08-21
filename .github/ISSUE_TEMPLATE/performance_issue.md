---
name: Performance Issue
about: Report performance problems with relay
title: '[PERFORMANCE] '
labels: ['performance', 'needs-triage']
assignees: ['howmanysmall']
---

## Performance Issue Description

A clear description of the performance problem you're experiencing.

## Environment

- **OS**: [e.g. macOS 14.1, Ubuntu 22.04, Windows 11]
- **Relay Version**: [run `relay --version`]
- **Hardware**: [CPU, RAM, Storage type (SSD/HDD/Network)]
- **File System**: [NTFS, ext4, APFS, etc.]

## Performance Data

### Dataset Information
- **Number of files**: 
- **Total size**: 
- **Average file size**: 
- **File types**: [e.g. text files, images, videos, etc.]
- **Directory depth**: 

### Current Performance
- **Time taken**: 
- **Transfer speed**: 
- **CPU usage**: [if known]
- **Memory usage**: [if known]
- **Disk I/O**: [if known]

### Expected Performance
What performance did you expect to see?

## Command Used

```bash
relay mirror ./source ./dest --verbose --profile performance
```

## Configuration

```jsonc
// relay.jsonc - performance-related settings
{
  "profiles": {
    "performance": {
      // your config
    }
  }
}
```

## Comparison Data

If you've compared with other tools (rsync, robocopy, etc.), please share:

| Tool | Time | Speed | Notes |
|------|------|-------|-------|
| relay | 5m 30s | 50 MB/s | Current performance |
| rsync | 3m 15s | 80 MB/s | Expected benchmark |

## System Monitoring

If you have system monitoring data during the operation, please include:
- CPU usage graphs
- Memory usage
- Disk I/O statistics
- Network usage (if applicable)

## Profiling Data

If you've run any profiling tools, please attach the results:
- Go pprof output
- System profiling data
- Custom benchmarks

## Additional Context

- Are you syncing to/from network drives?
- Are there any antivirus or security software running?
- Is this reproducible consistently?
- Does the performance degrade over time?