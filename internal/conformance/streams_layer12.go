// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: Apache-2.0

package conformance

// layer12Streams lists the Layer I and Layer II conformance streams, with the checksums ISO
// publishes.
//
// ISO publishes them inside MP4 files. Each has one reference that covers its whole decoded
// output, with one channel for the mono streams l1_fl4, l2_fl13 and l2_test30. l2_test33 holds
// MPEG-2 Layer I frames despite its name. l2_test27 has no reference, so it is not listed.
var layer12Streams = []Stream{
	{
		Name: "l1_fl1", MD5: "36e89640cdabc4da7f3d3b5f0e785422", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl1", MD5: "3e5d7738dda2585102809a61d8a323b7",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l1_fl2", MD5: "21af4701ed5257ac9aa7da808ba2db77", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl2", MD5: "6a3cba2bb379093e47725ba7f65f5113",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l1_fl3", MD5: "74c98a65c73e747a1324ada50b1b538d", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl3", MD5: "6b38befe4a840ced9c2d372bf469d43c",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l1_fl4", MD5: "4c8d0f082c426051fbecd614db6f0d9e", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl4", MD5: "af9a4e2450693eb57b15137b951ce7e7",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l1_fl5", MD5: "cfeefcc015e2d5c70bcebadf34ed9ed2", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl5", MD5: "90dac8170e1d2b167b7a56f8b8f39273",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l1_fl6", MD5: "b267ddc517707bac577a8d919f7b6615", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl6", MD5: "a035885383b6dd79551d03561835a5ea",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l1_fl7", MD5: "2c0d01683740e1f29f8ba9f15aead006", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl7", MD5: "d0349da7432a77438c8cc48cc047fab2",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l1_fl8", MD5: "8350a7b8cfa49c7bee8119591be9168d", MP4: true,
		References: []Reference{
			{
				Name: "l1_fl8", MD5: "cce88443c16d9c1b5c1646966223207f",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_fl10", MD5: "4063619f04c649d223a906c4bfc2b6f8", MP4: true,
		References: []Reference{
			{
				Name: "l2_fl10", MD5: "5c2365aba1e756a8f7134f24a54250aa",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_fl11", MD5: "09c21bb71d5c8a831bad77cf42986a1f", MP4: true,
		References: []Reference{
			{
				Name: "l2_fl11", MD5: "ab9937a5f70dc79e6e054c289824a7d2",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_fl12", MD5: "5eaa20420f63c23596e978141eb0814a", MP4: true,
		References: []Reference{
			{
				Name: "l2_fl12", MD5: "3b947c11f20ce219aa774dce4940535b",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_fl13", MD5: "2bc0d1412d72b59328e35901104a8aff", MP4: true,
		References: []Reference{
			{
				Name: "l2_fl13", MD5: "acc5a576ff515031afba4db9590f8b4d",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_fl14", MD5: "ddf53f9622182b6c295efc38925f2332", MP4: true,
		References: []Reference{
			{
				Name: "l2_fl14", MD5: "45aa9df1731f91a6f66f27ce62fc0336",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_fl15", MD5: "f5aceb85c891114bc207ac372564a68e", MP4: true,
		References: []Reference{
			{
				Name: "l2_fl15", MD5: "f86988fde021e4f5fe905e4abe1d3825",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_fl16", MD5: "00fb712e9eb85b60c6a7d58f0975d409", MP4: true,
		References: []Reference{
			{
				Name: "l2_fl16", MD5: "9fb96cbbf883168dadc9675ef6a41b25",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test24", MD5: "257713013a777823e1250953a2fa026f", MP4: true,
		References: []Reference{
			{
				Name: "l2_test24", MD5: "c185cc51397f4667a413b8da2924cc8d",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test25", MD5: "b7fdbb789292aafabeb18dd15e0ce501", MP4: true,
		References: []Reference{
			{
				Name: "l2_test25", MD5: "a9eca4871f3fea213c19f7070fca852d",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test26", MD5: "2210bfe3261cb0bf901014f9e102fe9a", MP4: true,
		References: []Reference{
			{
				Name: "l2_test26", MD5: "ffa87f41e22771f5dce56717a7439147",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test28", MD5: "2210bfe3261cb0bf901014f9e102fe9a", MP4: true,
		References: []Reference{
			{
				Name: "l2_test28", MD5: "9ff2039d71c53fc80426d369892b1703",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test29", MD5: "7bd738fb1b09778c6a2dc8fcb0c36b42", MP4: true,
		References: []Reference{
			{
				Name: "l2_test29", MD5: "0cc32548a10a0d501d60509c810b061d",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test30", MD5: "56389f71761899a3c2431f46c5483bd2", MP4: true,
		References: []Reference{
			{
				Name: "l2_test30", MD5: "6aeeb3d43dad61570da52209aad8e9f4",
				Start: 0, Channels: []int{0}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test31", MD5: "35dd50d0fb4707aa5a19d8479e032fe5", MP4: true,
		References: []Reference{
			{
				Name: "l2_test31", MD5: "7cd13e8e39ae20bafa53e7f9e441e5ac",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test32", MD5: "a15a4105b2f208423ab64836d05ffdf7", MP4: true,
		References: []Reference{
			{
				Name: "l2_test32", MD5: "730165bf995401ac3f5ee8aac058729e",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test33", MD5: "772a583dddba845cd93a5fa1f046d41d", MP4: true,
		References: []Reference{
			{
				Name: "l2_test33", MD5: "b7dbacf412212b4e6d1c2e193178d85c",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
	{
		Name: "l2_test34", MD5: "ea77fd0c930c769cd2a93c6a14f4deb4", MP4: true,
		References: []Reference{
			{
				Name: "l2_test34", MD5: "0e039271f61f286ca6b0a07a35508b3b",
				Start: 0, Channels: []int{0, 1}, Alternative: false,
			},
		},
	},
}
