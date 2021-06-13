package megacli

import (
	_ "fmt"
	"testing"

	"github.com/influxdata/telegraf/testutil"
)

func TestGatherRaid(t *testing.T) {
	m := Megacli{
		PathMegacli: "/usr/bin/MegaCli",
		Timeout: 10,
	}
	// overwriting exec commands with mock commands
	var acc testutil.Accumulator

	mockData := `

Adapter 0 -- Virtual Drive Information:
Virtual Drive: 0 (Target Id: 0)
Name                :
RAID Level          : Primary-5, Secondary-0, RAID Level Qualifier-3
Size                : 164.0 GB
Sector Size         : 512
Is VD emulated      : Yes
Parity Size         : 82.0 GB
State               : Optimal
Strip Size          : 64 KB
Number Of Drives    : 3
Span Depth          : 1
Default Cache Policy: WriteBack, ReadAdaptive, Cached, Write Cache OK if Bad BBU
Current Cache Policy: WriteBack, ReadAdaptive, Cached, Write Cache OK if Bad BBU
Default Access Policy: Read/Write
Current Access Policy: Read/Write
Disk Cache Policy   : Disk's Default
Encryption Type     : None
Default Power Savings Policy: Controller Defined
Current Power Savings Policy: None
Can spin up in 1 minute: No
LD has drives that support T10 power conditions: No
LD's IO profile supports MAX power savings with cached writes: No
Bad Blocks Exist: No
Is VD Cached: No


Number of Dedicated Hot Spares: 1
    0 : EnclId - 32 SlotId - 3 

Exit Code: 0x00
`
	err := m.gatherRaidStatus(string(mockData), &acc)
	if err != nil {
		t.Fatal(err)
	}

        raidtags := map[string]string {
                "adapter":         "0",
                "level_primary":   "5",
                "level_qualifier": "3",
                "level_secondary": "0",
                "virtual_drive":   "0",
        }
        raidfields := map[string]interface{} {
                "cache_policy":     int64(1),
                "drives_cnt":       int64(3),
                "exist_bad_blocks": int64(0),
                "hotspares_num":    int64(1),
                "parity_size":      uint64(88046829568),
                "sector_size":      int64(512),
                "size":             uint64(176093659136),
                "state":            int64(1),
                "strip_size":       uint64(65536),
        }
        acc.AssertContainsTaggedFields(t, "megacli_raid", raidfields, raidtags)
}

func TestGatherDisk(t *testing.T) {
	m := Megacli{
		PathMegacli: "/usr/bin/MegaCli",
		Timeout: 10,
	}
	// overwriting exec commands with mock commands
	var acc testutil.Accumulator

	mockData := `
                                     
Adapter #0

Enclosure Device ID: 32
Slot Number: 0
Drive's position: DiskGroup: 0, Span: 0, Arm: 0
Enclosure position: 1
Device Id: 0
WWN: 55cd2e415316012a
Sequence Number: 2
Media Error Count: 0
Other Error Count: 0
Predictive Failure Count: 0
Last Predictive Failure Event Seq Number: 0
PD Type: SATA

Raw Size: 894.252 GB [0x6fc81ab0 Sectors]
Non Coerced Size: 893.752 GB [0x6fb81ab0 Sectors]
Coerced Size: 893.75 GB [0x6fb80000 Sectors]
Sector Size:  512
Logical Sector Size:  512
Physical Sector Size:  4096
Firmware state: Online, Spun Up
Device Firmware Level: DL69
Shield Counter: 0
Successful diagnostics completion on :  N/A
SAS Address(0): 0x4433221104000000
Connected Port Number: 2(path0) 
Inquiry Data: PHYG045501DG960CGN  SSDSC2KG960G8R                          XCV1DL69
FDE Capable: Not Capable
FDE Enable: Disable
Secured: Unsecured
Locked: Unlocked
Needs EKM Attention: No
Foreign State: None 
Device Speed: 6.0Gb/s 
Link Speed: 6.0Gb/s 
Media Type: Solid State Device
Drive Temperature :22C (71.60 F)
PI Eligibility:  No 
Drive is formatted for PI information:  No
PI: No PI
Drive's NCQ setting : N/A
Port-0 :
Port status: Active
Port's Linkspeed: 6.0Gb/s 
Drive has flagged a S.M.A.R.T alert : No



Exit Code: 0x00
`
	err := m.gatherDiskStatus(string(mockData), &acc)
	if err != nil {
		t.Fatal(err)
	}

        disktags := map[string]string{
                "adapter":     "0",
                "en_deviceid": "32",
                "en_position": "1",
                "slot":        "0",
                "type":        "SATA",
        }   
        diskfields := map[string]interface{}{
                "error_media_cnt":      int64(0),
                "error_other_cnt":      int64(0),
                "error_prefailure_cnt": int64(0),
                "islocked":             int64(0),
                "last_prefailure_seq":  int64(0),
                "port_status":          int64(1),
                "size_gb":              float64(894.252),
                "state":                int64(1),
        }   
        acc.AssertContainsTaggedFields(t, "megacli_disk", diskfields, disktags)

}

func TestGatherBbu(t *testing.T) {
	m := Megacli{
		PathMegacli: "/usr/bin/MegaCli",
		Timeout: 10,
	}
	// overwriting exec commands with mock commands
	var acc testutil.Accumulator

	mockData := `
BBU status for Adapter: 0

BatteryType: BBU
Voltage: 3892 mV
Current: 0 mA
Temperature: 25 C
Battery State: Optimal
BBU Firmware Status:

  Charging Status              : None
  Voltage                                 : OK
  Temperature                             : OK
  Learn Cycle Requested                   : No
  Learn Cycle Active                      : No
  Learn Cycle Status                      : OK
  Learn Cycle Timeout                     : No
  I2c Errors Detected                     : No
  Battery Pack Missing                    : No
  Battery Replacement required            : No
  Remaining Capacity Low                  : No
  Periodic Learn Required                 : No
  Transparent Learn                       : No
  No space to cache offload               : No
  Pack is about to fail & should be replaced : No
  Cache Offload premium feature required  : No
  Module microcode update required        : No

BBU GasGauge Status: 0x0128 
Relative State of Charge: 90 %
Charger Status: Complete
Remaining Capacity: 333 mAh
Full Charge Capacity: 372 mAh
isSOHGood: Yes

Exit Code: 0x00
`
	err := m.gatherBbuStatus(string(mockData), &acc)
	if err != nil {
		t.Fatal(err)
	}

	bbutags := map[string]string {
		"adapter":   "0",
		"type":      "BBU",
	}
	bbufields := map[string]interface{} {
		"charge_relative":   int64(90),
		"charge_status":     int64(1),
		"issohgood":         int64(1),
		"state":             int64(1),
	}
	acc.AssertContainsTaggedFields(t, "megacli_bbu", bbufields, bbutags)
}
