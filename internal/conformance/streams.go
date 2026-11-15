// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformance

// Where references start later than their streams' decoded output.
const (
	// stereoRun is the first sample of the run of 100 stereo frames of l3_10201, after 10 mono
	// frames of 1152.
	stereoRun = 11520
	// monoRun is the first sample of the last run of 18 mono frames of l3_10201, after 110 frames
	// of 1152.
	monoRun = 126720
	// joined is the first sample of l3_10204 after its first 2 frames of 1152, whose main data
	// lies before the stream starts and which the reference leaves out.
	joined = 2304
)

// streams lists the Layer III conformance streams, with the checksums ISO publishes.
//
// Most streams have one reference that covers the whole decoded output. The reference of
// l3_10204, which starts midway, leaves out its first 2 frames. The references of
// l3_10201 cover its runs of mono, stereo and mono frames in turn, and the stereo run has two
// alternatives, named AA and noAA, which differ over its last 19 frames. The l3_10501 streams each
// carry one or two channels of a multichannel program, whose references hold one channel each.
var streams = []Stream{
	{
		Name: "l3_10101", MD5: "b97e00a879f5a8f3f74f213a4a6a8b75",
		References: []Reference{
			{
				Name: "l3_10101", MD5: "4a0a6ab5347895e0a27c3d310d18882a",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10102", MD5: "159e881b8040dbc1834ee67d864cea24",
		References: []Reference{
			{
				Name: "l3_10102", MD5: "3b6b7da9bc0ca0088c9d08ebc7b4c986",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10103", MD5: "349da5170f057cf1cec128c12f8fee06",
		References: []Reference{
			{
				Name: "l3_10103", MD5: "f594442ae2ea946b098977250ca78ebd",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10104", MD5: "835df4599d2b3500679a5f3a3e83d93f",
		References: []Reference{
			{
				Name: "l3_10104", MD5: "21e02ee1d3b21975c25b8bc0eb6accc1",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10105", MD5: "6d8b732871fca1fc5aee122a6f7ec8d3",
		References: []Reference{
			{
				Name: "l3_10105", MD5: "5aae31b3c1ba445393d2ad46b0fe8ed0",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10106", MD5: "ace39dd25755ba34620c4f1dc0d47a73",
		References: []Reference{
			{
				Name: "l3_10106", MD5: "82d31f324a75389a03d84e6ecd7f77e0",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10107", MD5: "d9c7bf1e670261869bb0a43924e1f95a",
		References: []Reference{
			{
				Name: "l3_10107", MD5: "aeee04867a2126f3bbb5e1235cfc0460",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10201", MD5: "c132e8a00002de6aadf66f0feeb28e12",
		References: []Reference{
			{
				Name: "l3_10201_0", MD5: "1e248c943bb9ebf57aff17f728615a94",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
			{
				Name: "l3_10201_1_AA", MD5: "9a33b96d1ed19d5e56e0cb47c8820245",
				Start: stereoRun, Channels: []int{0, 1}, Alternative: false,
			},
			{
				Name: "l3_10201_1_noAA", MD5: "4f90a0bbc9208acc4b3bb215bf58823d",
				Start: stereoRun, Channels: []int{0, 1}, Alternative: true,
			},
			{
				Name: "l3_10201_2", MD5: "b63a1c6d2bc5c1a9d1f1a566438061ea",
				Start: monoRun, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10202", MD5: "febed1da65a1b7001617b01fd9422dba",
		References: []Reference{
			{
				Name: "l3_10202", MD5: "7b8d582f8e5e9bc8cc2d83098a1bce6b",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10203", MD5: "30a992db0c956fceaf3625bd899ba31b",
		References: []Reference{
			{
				Name: "l3_10203", MD5: "2ec87b2c1439783cdab425d9c31e991f",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10204", MD5: "bac6c449e5dc91898b31ef423938162e",
		References: []Reference{
			{
				Name: "l3_10204", MD5: "35d2440de752309adf8df7783f69084e",
				Start: joined, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10205", MD5: "19e61d11534a5e3b01f0f20b77f19ea4",
		References: []Reference{
			{
				Name: "l3_10205", MD5: "ecc7839119466b45ea0dfbeb7103f698",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10206", MD5: "0b19adac6341e05367e45233152e441f",
		References: []Reference{
			{
				Name: "l3_10206", MD5: "53bbc132a0498ed3700b806d575437d9",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10207", MD5: "717b8d33353e8a7b757d196ff4ecf6d7",
		References: []Reference{
			{
				Name: "l3_10207", MD5: "94de75c6f8df8fabea294ae07f914a46",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10501_b01+b02", MD5: "b4ecca101ab6a2ec3dc6abfc72b259b1",
		References: []Reference{
			{
				Name: "l3_10501_b00", MD5: "29c9fe157f0c468f27baeebfcbdb9090",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
			{
				Name: "l3_10501_b01", MD5: "64074961dafd69356d87d364eae2db08",
				Start: 0, Channels: []int{1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10501_b01_b02", MD5: "b4ecca101ab6a2ec3dc6abfc72b259b1",
		References: []Reference{
			{
				Name: "l3_10501_b00", MD5: "29c9fe157f0c468f27baeebfcbdb9090",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
			{
				Name: "l3_10501_b01", MD5: "64074961dafd69356d87d364eae2db08",
				Start: 0, Channels: []int{1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10501_f00", MD5: "d9c7bf1e670261869bb0a43924e1f95a",
		References: []Reference{
			{
				Name: "l3_10501_f00", MD5: "e5af757269ea7663fc51c969939226d1",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_10501_f01+f02", MD5: "717b8d33353e8a7b757d196ff4ecf6d7",
		References: []Reference{
			{
				Name: "l3_10501_f01", MD5: "86f56a7e066fbd713d176edb548f4c14",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
			{
				Name: "l3_10501_f02", MD5: "38de1a6e0e07cfb50edccad0e7c1e721",
				Start: 0, Channels: []int{1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20140", MD5: "68d45e0cc4808b2ae5bde110425e33cd",
		References: []Reference{
			{
				Name: "l3_20140", MD5: "2a65c1c5dbff9445b5c3c713f6a54e4e",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20141", MD5: "71c813a7b4d62805407af733a5efea33",
		References: []Reference{
			{
				Name: "l3_20141", MD5: "2588d39a87aac0e69ea7ec2268691c13",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20142", MD5: "6031f84c5af0105912117fe87fd297e2",
		References: []Reference{
			{
				Name: "l3_20142", MD5: "1ea781584de8be4b771499b11a710dd8",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20143", MD5: "90318be5767dcd1783c9d0a818e867ab",
		References: []Reference{
			{
				Name: "l3_20143", MD5: "0e752085d2a1ec6bd731cfbcce7899f1",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20244", MD5: "84bfca3fa6e8440caaa5755119963afc",
		References: []Reference{
			{
				Name: "l3_20244", MD5: "bf8d932c563acc17bc784c654df6dac6",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20245", MD5: "b9afeac7a9eb70d43f7736b9d3b27eee",
		References: []Reference{
			{
				Name: "l3_20245", MD5: "51284f5e6e2b595a3288f52a41f486f6",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20246", MD5: "c36dd070237b29d0d4bd04850b832d03",
		References: []Reference{
			{
				Name: "l3_20246", MD5: "0546fd492593caa3fc41479a2c5d2df4",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20247", MD5: "87035ca30d0a2c3890139811260d1a0e",
		References: []Reference{
			{
				Name: "l3_20247", MD5: "3cf1fe305c4657c701b33756c58fadf1",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_20248", MD5: "822ce9f053e43e95551d279622fca520",
		References: []Reference{
			{
				Name: "l3_20248", MD5: "e863ab2e94dc6264f537301870f961fa",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25101", MD5: "6398e08473f34a14aac9180d8019d940",
		References: []Reference{
			{
				Name: "l3_25101", MD5: "7255d113a9a589edf4375135772e5a77",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25102", MD5: "acbfe159b1a6b44e1bd1c6304590b37a",
		References: []Reference{
			{
				Name: "l3_25102", MD5: "3ad2e09d05886579d6aa09575627deb8",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25103", MD5: "a7612ef0e9447034528fbc859d9909be",
		References: []Reference{
			{
				Name: "l3_25103", MD5: "62e5538171b64d3bb6f8c32f90b0e328",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25104", MD5: "f56f493343b1002b9cb4201e23b7c323",
		References: []Reference{
			{
				Name: "l3_25104", MD5: "4d4e94c9510355b345142532f3e92c5d",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25201", MD5: "83d9622698606739b6537c81c6fd78e0",
		References: []Reference{
			{
				Name: "l3_25201", MD5: "69493e084e17bb46a4ab2ad5a1e78695",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25202", MD5: "b349d9e9adaf1b7d56936d266ed40822",
		References: []Reference{
			{
				Name: "l3_25202", MD5: "1bf3711a2cadf9a0bc0b25c375c711df",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25203", MD5: "fd03347ff01ac6e11516d6e85174782a",
		References: []Reference{
			{
				Name: "l3_25203", MD5: "aca6064b9e45e043bd508a8e20ae0456",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25204", MD5: "ace2f87a04ceb336e28bac5e16beafe3",
		References: []Reference{
			{
				Name: "l3_25204", MD5: "123a4320bfc04e8fb8f1f976906a575d",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25205", MD5: "fe3c3abe6b4b7711bcdc3b53840c0a71",
		References: []Reference{
			{
				Name: "l3_25205", MD5: "a31819e60de044ad8fa1788b3e92c184",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25206", MD5: "3b932cafffdd11331daf6074fee175e7",
		References: []Reference{
			{
				Name: "l3_25206", MD5: "5504d525c2b3e5a9bc1dfd5cb17c6238",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25207", MD5: "fac21fb4af2f707fcb9d9481e7357a43",
		References: []Reference{
			{
				Name: "l3_25207", MD5: "d296ea917e3c8c73011e722302180afc",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25208", MD5: "eb767fc8e6a0efe874e568c45e610796",
		References: []Reference{
			{
				Name: "l3_25208", MD5: "7977c26e8650c86fafe4e5fae8104af6",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l3_25209", MD5: "25d92229fc191352aa1bdd8c9da57553",
		References: []Reference{
			{
				Name: "l3_25209", MD5: "0f98b45b927c32e92b186025aa88242d",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
}
