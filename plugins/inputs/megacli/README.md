# MegaCli Input Plugin

Get standard MegaCli(`ver-8.0.x`) metrics, requires MegaCli executable.

### Configuration:

```toml
# Read megacli's basic status information
[[inputs.megacli]]
  ## Optionally specify the path to the megacli executable
  path_megacli = "/usr/bin/MegaCli"

  ## Gather info of the following type:
  ## raid, disk, bbu
  ## default is gather all of the three type
  #gather_type = ["raid", "disk", "bbu"]
  gather_type = ['disk', 'raid', 'bbu']

  ## On most platforms used cli utilities requires root access.
  ## Setting 'use_sudo' to true will make use of sudo to run MegaCli.
  ## Sudo must be configured to allow the telegraf user to run MegaCli
  ## without a password.
  use_sudo = false

  ## Timeout for the cli command to complete.
  timeout = "3s"
```

### tags & Fields:

#### raid
```
tags:
  adapter:          raid adapter number
  level_primary:    raid level, for Primary
  level_secondary:  raid level, for Seconday
  level_qualifier:  raid level, for Qualifier
  virtual_drive:    raid virtual drive, one adapter maybe have multiple virtual device

fields:
  cache_policy: (int64)
     WriteBack:    1
     WriteThrough: 2

  derives_cnt:  (int64)    # physical disk number, exclude hotspare.
  hostspare_num:(int64)    # hostspare disk number for current virtual device.

  exist_bad_blocks:(int)
     Yes: 1
     No:  0

  state: (int64)
    Optimal:  1
    Degraded: 2
    Other:    0

  size: (int64)        # virtual device total size, in bytes
  sector_size: (int64) # virtual device sector size, in bytes
  stripe_size: (int64) # virtual device stripe size, in bytes

```

#### disk

```
tags:
  adapter:       raid adapter number
  en_deviceid:   Enclosure Device ID
  slot:          Slot Number
  en_position:   Enclosure position
  type:          PD Type

fields:
  error_media_cnt(int64):      Media Error Count
  error_other_cnt(int64):      Other Error Count
  error_prefailure_cnt(int64): Predictive Failure Count
  error_prefailure_seq(int64): Last Predictive Failure Event Seq Number
  size_gb(float64):              disk raw size in GB

  port_status:          Port status
    Active: 1
    Other:  0

  state:
    Failed:      -1
    Unconfigured: 0
    Online:       1
    Rebuild:      2
    Hotspare:     3
```

#### bbu

```
tags:
  adapter:       raid adapter number
  type:          BatteryType

fields:
  state:         Battery State
    Optimal:  1
    Other:    0

  charge_status: Charge Status
    Complete: 1
    Other:    0

  charge_relative(int64): Relative State of Charge, in percent(%)

  issohgood(int64):       isSOHGood
     Yes:  1
     No:   0
```


### Example Output:

```
$ telegraf --config telegraf.conf --input-filter megacli --test
> megacli_disk,adapter=0,en_deviceid=32,en_position=1,host=cz2,slot=0,type=SATA error_media_cnt=0i,error_other_cnt=0i,error_prefailure_cnt=0i,islocked=0i,last_prefailure_seq=0i,port_status=1i,size_gb=745.211,state=1i 1623396755000000000
> megacli_disk,adapter=0,en_deviceid=32,en_position=1,host=cz2,slot=1,type=SATA error_media_cnt=0i,error_other_cnt=0i,error_prefailure_cnt=0i,islocked=0i,last_prefailure_seq=0i,port_status=1i,size_gb=745.211,state=1i 1623396755000000000
> megacli_disk,adapter=0,en_deviceid=32,en_position=1,host=cz2,slot=2,type=SATA error_media_cnt=0i,error_other_cnt=0i,error_prefailure_cnt=0i,islocked=0i,last_prefailure_seq=0i,port_status=1i,size_gb=745.211,state=1i 1623396755000000000
> megacli_disk,adapter=0,en_deviceid=32,en_position=1,host=cz2,slot=3,type=SATA error_media_cnt=0i,error_other_cnt=0i,error_prefailure_cnt=0i,islocked=0i,last_prefailure_seq=0i,port_status=1i,size_gb=745.211,state=3i 1623396755000000000
> megacli_raid,adapter=0,host=cz2,level_primary=5,level_qualifier=3,level_secondary=0,virtual_drive=0 cache_policy=1i,drives_cnt=3i,exist_bad_blocks=0i,hotspares_num=1i,parity_size=88046829568i,sector_size=512i,size=176093659136i,state=1i,strip_size=65536i 1623396755000000000
> megacli_raid,adapter=0,host=cz2,level_primary=5,level_qualifier=3,level_secondary=0,virtual_drive=1 cache_policy=1i,drives_cnt=3i,exist_bad_blocks=0i,hotspares_num=1i,parity_size=871609925632i,sector_size=512i,size=1741626418397i,state=1i,strip_size=65536i 1623396755000000000
> megacli_bbu,adapter=0,host=cz2,type=BBU charge_relative=0i,charge_status=1i,issohgood=1i,state=1i 1623396755000000000
```




