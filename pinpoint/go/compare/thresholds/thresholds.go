// Package thresholds implements the thresholds for hypothesis testing.
//
// The "low threshold" is the traditional significance threshold. If the p-value is
// below the "low threshold", we say the two samples come from different
// distributions (reject the null hypothesis).
//
// We also define a "high threshold". If the p-value is above the "high threshold",
// we say the two samples come from the same distribution (reject the alternative
// hypothesis). The high thresholds listed in this script are hard coded. They
// are generated via [thresholds_functional.py] and [thresholds_performance.py]
//
// If the p-value is in between the two thresholds, we fail to reject either
// hypothesis. This means we need more information to make a decision. As the
// sample sizes increase, the high threshold decreases until it crosses the
// low threshold. This way, there's a limit on the number of repeats.
// Thresholds are used in bisection to reduce the number of computing
// resources needed to run bisection. The high threshold scales depending
// on the normalized magnitude.
//
// The normalized magnitude is the size of the difference normalized by the
// larger interquartile range of the two samples.
// The greater the perceived difference, the fewer iterations are necessary
// for concluding that two samples come from the same population.
// This [document] gives more context for the high thresholds.
//
// On the other hand, the low threshold is statically
// defined, since the burden of significance is the same regardless
// of sample size and normalized magnitude.
//
// [thresholds_functional.py]: https://chromium.googlesource.com/catapult.git/+/f25d23e77a963e88af9199c7c3a0638268e44538/dashboard/dashboard/pinpoint/models/compare/thresholds_functional.py
// [thresholds_performance.py]: https://chromium.googlesource.com/catapult.git/+/f25d23e77a963e88af9199c7c3a0638268e44538/dashboard/dashboard/pinpoint/models/compare/thresholds_performance.py
// [document]: https://docs.google.com/document/d/19gig8pv8ei2y45mVDp5awZGj39bZoFjNDn65uOfyUUY/edit#bookmark=id.8k02cmyj7ral

package thresholds

// The low threshold used in hypothesis testing.
// Note that this implies we could be rejecting the null hypothesis if the
// p-value is less than this value. It can happen that because we're generating
// the table of "high" thresholds with randomised permutations, that this value
// is actually higher than the "high" thresholds.
const LowThreshold = 0.01

type thresholds [][]float64

// HighThresholdPerformance returns the high threshold for performance hypothesis
// testing given the normalized_magnitude and the sample_size.
//
// The normalized magnitude is an estimate of the size of differences to look for,
// normalized by the interquartile range (IQR). We need more values to find
// smaller differences.
func HighThresholdPerformance(normalized_magnitude float64, sample_size int) (float64, error) {
	magnitude_index := int(normalized_magnitude*10) - 3
	return getHighThreshold(highThresholdsPerformance, magnitude_index, sample_size), nil
}

// HighThresholdFunctional returns the high threshold for functional hypothesis
// testing given the normalized_magnitude and the sample_size.
//
// The normalized magnitude is an estimate of failure rate between 0 and 1.
// We need more values to find smaller differences.
func HighThresholdFunctional(normalized_magnitude float64, sample_size int) (float64, error) {
	magnitude_index := int(normalized_magnitude*10) - 1
	return getHighThreshold(highThresholdsFunctional, magnitude_index, sample_size), nil
}

func getHighThreshold(high_thresholds thresholds, magnitude_index int, sample_size int) float64 {
	magnitude_index = max(magnitude_index, 0)
	magnitude_index = min(magnitude_index, len(high_thresholds)-1)
	sample_size_index := min(sample_size, len(high_thresholds[magnitude_index])) - 1
	return high_thresholds[magnitude_index][sample_size_index]
}

// highThresholdsFunctional is the set of high thresholds
// used in functional analysis by normalized_magnitude.
// Run [thresholds_functional.py] to generate these numbers.
// The magnitudes are expressed in difference in failure rate.
// The sample sizes start at 1.
//
// [thresholds_functional.py]: https://chromium.googlesource.com/catapult.git/+/f25d23e77a963e88af9199c7c3a0638268e44538/dashboard/dashboard/pinpoint/models/compare/thresholds_functional.py
var highThresholdsFunctional = thresholds{
	// normalized magnitude: 0.1
	{
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
		1.000, 1.000, 1.000, .3285, .3282, .3280, .3278, .3275, .3273, .3271,
		.3269, .3268, .3266, .3264, .3262, .3261, .3259, .3258, .3256, .3255,
		.3254, .3252, .3251, .1590, .1589, .1589, .1589, .1589, .1589, .1588,
		.1588, .1588, .1588, .1587, .1587, .1587, .1587, .1587, .1587, .1586,
		.0827, .0827, .0827, .0827, .0827, .0827, .0827, .0827, .0827, .0827,
		.0827, .0827, .0827, .0828, .0828, .0828, .0444, .0444, .0444, .0445,
		.0445, .0445, .0445, .0445, .0445, .0445, .0445, .0445, .0445, .0446,
		.0446, .0446, .0244, .0244, .0244, .0244, .0244, .0244, .0244, .0244,
		.0244, .0244, .0244, .0245, .0245, .0245, .0135, .0135, .0135, .0135,
		.0136, .0136, .0136, .0136, .0136, .0136, .0136, .0136, .0136, .0136,
		.0136, .0076,
	},
	// normalized magnitude: 0.2
	{
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
		.3410, .3399, .3389, .3379, .3371, .3363, .3356, .3350, .3343, .3338,
		.1607, .1606, .1605, .1604, .1603, .1602, .1601, .1601, .0819, .0819,
		.0820, .0820, .0820, .0821, .0821, .0821, .0432, .0433, .0433, .0433,
		.0434, .0434, .0435, .0435, .0233, .0233, .0233, .0234, .0234, .0234,
		.0235, .0127, .0127, .0127, .0128, .0128, .0128, .0128, .0070,
	},
	// normalized magnitude: 0.3
	{
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000,
		1.000, 1.000, .3560, .3532, .3507, .3486, .3467, .3450, .3435, .1625,
		.1623, .1620, .1618, .1616, .0810, .0811, .0812, .0813, .0814, .0418,
		.0419, .0421, .0422, .0423, .0220, .0221, .0222, .0223, .0224, .0118,
		.0118, .0119, .0120, .0063,
	},
	// normalized magnitude: 0.4,
	{
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, 1.000, .3682,
		.3634, .3594, .3560, .1647, .1642, .1638, .1634, .0800, .0802, .0804,
		.0806, .0404, .0406, .0409, .0207, .0209, .0210, .0212, .0108, .0109,
		.0110, .0057,
	},
	// normalized magnitude: 0.5,
	{
		1.000, 1.000, 1.000, 1.000, 1.000, 1.000, .3914, .3816, .3741, .3682,
		.1666, .1659, .1652, .0789, .0793, .0796, .0388, .0392, .0192, .0195,
		.0198, .0098,
	},
	// normalized magnitude: 0.6,
	{
		1.000, 1.000, 1.000, 1.000, 1.000, .4047, .3914, .1700, .1686, .1675,
		.0775, .0781, .0366, .0373, .0175, .0180, .0185, .0088,
	},
	// normalized magnitude: 0.7,
	{
		1.000, 1.000, 1.000, .4533, .4238, .4047, .1717, .1700, .0758, .0768,
		.0348, .0156, .0163, .0074,
	},
	// normalized magnitude: 0.8,
	{
		1.000, 1.000, .5050, .4533, .1771, .1740, .0730, .0746, .0322, .0137,
		.0147, .0064,
	},
	// normalized magnitude: 0.9,
	{
		1.000, .6171, .5050, .1814, .0668, .0706, .0279, .0110, .0043,
	},
	// normalized magnitude: 1.0,
	{
		1.000, .1940, .0469, .0132, .0040,
	},
}

// highThresholdsPerformance is the set of high thresholds
// used in functional analysis by normalized magnitude.
// Run [thresholds_performance.py] to generate these numbers.
// The normalized magnitudes are expressed in multiples of the
// interquartile range (IQR).
// The sample sizes start at 1.
//
// [thresholds_performance.py]: https://chromium.googlesource.com/catapult.git/+/f25d23e77a963e88af9199c7c3a0638268e44538/dashboard/dashboard/pinpoint/models/compare/thresholds_performance.py
var highThresholdsPerformance = thresholds{
	// normalized magnitude 0.3
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		1.000,
		.9582,
		1.000,
		.9699,
		1.000,
		.9770,
		1.000,
		.9817,
		1.000,
		.9850,
		1.000,
		.9874,
		.9768,
		.9893,
		.9800,
		.9907,
		.9825,
		.9754,
		.9846,
		.9781,
		.9724,
		.9804,
		.9752,
		.9706,
		.9776,
		.9733,
		.9694,
		.9756,
		.9626,
		.9686,
		.9656,
		.9628,
		.9602,
		.9578,
		.9557,
		.9537,
		.9518,
		.9435,
		.9486,
		.9409,
		.9338,
		.9446,
		.9378,
		.9314,
		.9307,
		.9249,
		.9145,
		.9144,
		.9144,
		.9097,
		.9143,
		.9012,
		.9015,
		.8936,
		.8860,
		.8828,
		.8760,
		.8807,
		.8853,
		.8861,
		.8833,
		.8670,
		.8615,
		.8529,
		.8512,
		.8433,
		.8266,
		.8315,
		.8275,
		.8322,
		.8228,
		.8247,
		.8077,
		.8152,
		.8095,
		.8014,
		.7837,
		.7838,
		.7720,
		.7653,
		.7497,
		.7126,
		.7010,
		.6794,
		.6730,
		.6790,
		.6729,
		.6651,
		.6576,
		.6598,
		.6307,
		.6225,
		.6164,
		.6122,
		.5931,
		.5846,
		.5764,
		.5530,
		.5564,
		.5643,
		.5541,
		.5457,
		.5377,
		.5203,
		.5077,
		.4930,
		.5007,
		.5031,
		.4842,
		.4843,
		.4678,
		.4623,
		.4639,
		.4409,
		.4297,
		.4149,
		.4108,
		.4150,
		.3982,
		.3889,
		.3941,
		.3805,
		.3856,
		.3718,
		.3673,
		.3595,
		.3528,
		.3504,
		.3562,
		.3467,
		.3461,
		.3281,
		.3219,
		.3159,
		.3165,
		.3156,
		.3066,
		.3154,
		.3120,
		.3028,
		.2883,
		.2824,
		.2792,
		.2719,
		.2735,
		.2682,
		.2710,
		.2648,
		.2562,
		.2553,
		.2426,
		.2384,
		.2324,
		.2380,
		.2317,
		.2266,
		.2142,
		.2126,
		.1999,
		.1961,
		.1902,
		.1883,
		.1831,
		.1871,
		.1861,
		.1776,
		.1723,
		.1675,
		.1613,
		.1641,
		.1606,
		.1548,
		.1540,
		.1500,
		.1486,
		.1460,
		.1405,
		.1352,
		.1324,
		.1278,
		.1307,
		.1298,
		.1208,
		.1223,
		.1200,
		.1167,
		.1192,
		.1156,
		.1151,
		.1145,
		.1121,
		.1100,
		.1025,
		.1039,
		.1016,
		.0998,
		.1003,
		.0994,
		.0920,
		.0930,
		.0900,
		.0872,
		.0845,
		.0806,
		.0813,
		.0769,
		.0792,
		.0766,
		.0700,
		.0689,
		.0693,
		.0657,
		.0658,
		.0655,
		.0652,
		.0643,
		.0629,
		.0611,
		.0570,
		.0572,
		.0558,
		.0544,
		.0533,
		.0520,
		.0521,
		.0499,
		.0481,
		.0473,
		.0477,
		.0452,
		.0454,
		.0453,
		.0418,
		.0417,
		.0417,
		.0404,
		.0403,
		.0375,
		.0383,
		.0375,
		.0381,
		.0377,
		.0349,
		.0349,
		.0324,
		.0330,
		.0310,
		.0312,
		.0308,
		.0310,
		.0307,
		.0295,
		.0286,
		.0280,
		.0286,
		.0272,
		.0275,
		.0270,
		.0255,
		.0259,
		.0252,
		.0236,
		.0239,
		.0242,
		.0231,
		.0238,
		.0235,
		.0232,
		.0216,
		.0213,
		.0212,
		.0213,
		.0205,
		.0201,
		.0201,
		.0191,
		.0197,
		.0184,
		.0186,
		.0186,
		.0171,
		.0162,
		.0165,
		.0159,
		.0149,
		.0146,
		.0139,
		.0132,
		.0131,
		.0124,
		.0121,
		.0120,
		.0121,
		.0112,
		.0114,
		.0114,
		.0104,
		.0105,
		.0106,
		.0104,
		.0102,
		.0104,
		.0102,
		.0102,
		.0102,
		.0094,
		.0092,
		.0089,
		.0082,
		.0081,
		.0076,
		.0074,
		.0074,
		.0071,
		.0066,
		.0067,
		.0065,
		.0065,
		.0059,
		.0057,
		.0057,
		.0054,
		.0055,
		.0053,
		.0050,
		.0049,
		.0048,
		.0047,
		.0047,
		.0048,
		.0045,
		.0043,
		.0042,
		.0042,
		.0041,
		.0039,
		.0040,
		.0038,
		.0036,
		.0036,
		.0036,
		.0035,
		.0035,
		.0033,
		.0033,
		.0033,
		.0035,
		.0035,
		.0033,
		.0033,
		.0032,
		.0030,
		.0032,
		.0030,
	},
	// 0.4
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		1.000,
		.9582,
		1.000,
		.9699,
		1.000,
		.9770,
		1.000,
		.9817,
		1.000,
		.9850,
		.9726,
		.9874,
		.9768,
		.9677,
		.9599,
		.9720,
		.9650,
		.9589,
		.9536,
		.9344,
		.9311,
		.9282,
		.9257,
		.9235,
		.9104,
		.9198,
		.9082,
		.9170,
		.8972,
		.8792,
		.8627,
		.8476,
		.8338,
		.8286,
		.8167,
		.7988,
		.7757,
		.7607,
		.7408,
		.7283,
		.7167,
		.7278,
		.7171,
		.6970,
		.6733,
		.6561,
		.6492,
		.6340,
		.6283,
		.6025,
		.5826,
		.5715,
		.5503,
		.5444,
		.5356,
		.5306,
		.5133,
		.5062,
		.4763,
		.4541,
		.4336,
		.4220,
		.4284,
		.4176,
		.3984,
		.3849,
		.3702,
		.3584,
		.3572,
		.3427,
		.3117,
		.3167,
		.3083,
		.3004,
		.2838,
		.2786,
		.2694,
		.2622,
		.2554,
		.2366,
		.2310,
		.2234,
		.2174,
		.2087,
		.1956,
		.1976,
		.1856,
		.1833,
		.1726,
		.1691,
		.1658,
		.1626,
		.1582,
		.1519,
		.1427,
		.1324,
		.1305,
		.1202,
		.1193,
		.1158,
		.1134,
		.1157,
		.1101,
		.1058,
		.1049,
		.0960,
		.0981,
		.0962,
		.0913,
		.0854,
		.0815,
		.0770,
		.0749,
		.0772,
		.0758,
		.0710,
		.0684,
		.0660,
		.0666,
		.0611,
		.0591,
		.0577,
		.0570,
		.0530,
		.0504,
		.0520,
		.0500,
		.0502,
		.0471,
		.0481,
		.0450,
		.0448,
		.0410,
		.0397,
		.0386,
		.0357,
		.0339,
		.0330,
		.0315,
		.0310,
		.0300,
		.0293,
		.0275,
		.0273,
		.0257,
		.0251,
		.0240,
		.0228,
		.0212,
		.0198,
		.0194,
		.0194,
		.0188,
		.0176,
		.0171,
		.0164,
		.0153,
		.0150,
		.0131,
		.0130,
		.0117,
		.0122,
		.0117,
		.0115,
		.0108,
		.0098,
		.0098,
		.0092,
		.0087,
		.0082,
		.0082,
		.0079,
		.0073,
		.0073,
		.0067,
		.0065,
		.0063,
		.0063,
		.0057,
		.0053,
		.0052,
		.0052,
		.0050,
		.0049,
		.0046,
		.0043,
		.0041,
		.0038,
		.0037,
		.0038,
		.0035,
		.0034,
		.0033,
		.0032,
		.0032,
		.0029,
		.0028,
		.0026,
		.0025,
		.0026,
		.0024,
		.0025,
		.0023,
		.0022,
		.0023,
		.0022,
		.0021,
		.0020,
		.0018,
		.0018,
		.0017,
		.0016,
		.0015,
		.0015,
		.0014,
		.0014,
		.0013,
		.0012,
		.0011,
		.0012,
		.0011,
		.0011,
		.0010,
		.0010,
		.0009,
		.0009,
		.0009,
		.0009,
		.0008,
		.0008,
		.0008,
		.0007,
		.0007,
		.0006,
	},
	// 0.5
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		1.000,
		.9582,
		1.000,
		.9699,
		1.000,
		.9770,
		1.000,
		.9817,
		.9670,
		.9550,
		.9451,
		.9370,
		.9535,
		.9246,
		.9199,
		.9159,
		.8952,
		.8771,
		.8614,
		.8477,
		.8356,
		.7869,
		.8035,
		.7619,
		.7461,
		.7524,
		.7196,
		.6812,
		.6469,
		.6084,
		.5741,
		.5643,
		.5555,
		.5223,
		.4810,
		.4882,
		.4629,
		.4452,
		.4197,
		.4101,
		.3846,
		.3694,
		.3557,
		.3260,
		.3220,
		.2938,
		.2828,
		.2674,
		.2536,
		.2481,
		.2274,
		.2193,
		.2003,
		.1904,
		.1815,
		.1686,
		.1735,
		.1722,
		.1570,
		.1511,
		.1420,
		.1372,
		.1296,
		.1236,
		.1163,
		.1096,
		.1044,
		.0919,
		.0880,
		.0850,
		.0791,
		.0732,
		.0664,
		.0642,
		.0594,
		.0584,
		.0519,
		.0494,
		.0486,
		.0441,
		.0419,
		.0394,
		.0355,
		.0342,
		.0334,
		.0316,
		.0306,
		.0293,
		.0297,
		.0272,
		.0260,
		.0235,
		.0218,
		.0211,
		.0187,
		.0186,
		.0173,
		.0165,
		.0158,
		.0142,
		.0124,
		.0118,
		.0106,
		.0102,
		.0096,
		.0091,
		.0087,
		.0091,
		.0083,
		.0078,
		.0073,
		.0066,
		.0063,
		.0061,
	},
	// 0.6
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		1.000,
		.9582,
		1.000,
		.9699,
		1.000,
		.9770,
		.9592,
		.9086,
		.9339,
		.8951,
		.8905,
		.8619,
		.8610,
		.7972,
		.8014,
		.7693,
		.7418,
		.6877,
		.6555,
		.6277,
		.5918,
		.5719,
		.5442,
		.5012,
		.4556,
		.4401,
		.4118,
		.3739,
		.3355,
		.3135,
		.2894,
		.2917,
		.2674,
		.2542,
		.2389,
		.2255,
		.2012,
		.1888,
		.1727,
		.1587,
		.1487,
		.1360,
		.1232,
		.1169,
		.1025,
		.0955,
		.0916,
		.0774,
		.0759,
		.0727,
		.0681,
		.0601,
		.0547,
		.0512,
		.0453,
		.0418,
		.0382,
		.0338,
		.0308,
		.0298,
		.0286,
		.0269,
		.0246,
		.0207,
		.0201,
		.0190,
		.0179,
		.0155,
		.0140,
		.0122,
		.0117,
		.0105,
		.0098,
		.0097,
		.0087,
		.0081,
		.0068,
		.0061,
		.0058,
		.0051,
		.0048,
		.0046,
		.0040,
		.0037,
	},
	// 0.7
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		1.000,
		.9582,
		.9297,
		.9699,
		.8956,
		.8853,
		.8778,
		.8362,
		.8035,
		.7487,
		.7048,
		.6693,
		.6197,
		.5793,
		.5295,
		.4741,
		.4163,
		.4037,
		.3619,
		.3367,
		.2993,
		.2617,
		.2373,
		.2171,
		.1953,
		.1730,
		.1661,
		.1531,
		.1421,
		.1270,
		.1096,
		.0997,
		.0857,
		.0791,
		.0706,
		.0635,
		.0575,
		.0494,
		.0445,
		.0420,
		.0356,
		.0316,
		.0308,
		.0272,
		.0242,
		.0220,
		.0191,
		.0167,
		.0150,
		.0136,
		.0119,
		.0109,
		.0097,
		.0085,
		.0076,
		.0066,
		.0061,
		.0054,
		.0046,
		.0039,
		.0036,
		.0030,
		.0028,
		.0026,
	},
	// 0.8
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		1.000,
		.9582,
		.9297,
		.8502,
		.8439,
		.7951,
		.7197,
		.6295,
		.6187,
		.5341,
		.4910,
		.4383,
		.3655,
		.3235,
		.2908,
		.2649,
		.2270,
		.2048,
		.1745,
		.1561,
		.1369,
		.1139,
		.0993,
		.0850,
		.0761,
		.0649,
		.0561,
		.0505,
		.0446,
		.0357,
		.0324,
		.0281,
		.0240,
		.0197,
		.0176,
		.0152,
		.0126,
		.0108,
		.0092,
		.0079,
		.0070,
		.0063,
		.0052,
		.0041,
	},
	// 0.9
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		.8984,
		.8749,
		.7911,
		.7338,
		.6936,
		.6237,
		.5384,
		.4484,
		.4068,
		.3366,
		.2857,
		.2482,
		.2094,
		.1806,
		.1590,
		.1301,
		.1088,
		.0970,
		.0743,
		.0659,
		.0527,
		.0448,
		.0387,
		.0327,
		.0252,
		.0221,
		.0183,
		.0150,
		.0128,
		.0105,
		.0084,
		.0073,
		.0064,
		.0049,
	},
	// 1.0
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.9362,
		.8984,
		.7929,
		.6589,
		.5708,
		.4702,
		.4026,
		.3299,
		.2803,
		.2291,
		.1810,
		.1579,
		.1329,
		.1084,
		.0859,
		.0702,
		.0558,
		.0456,
		.0364,
		.0313,
		.0239,
		.0196,
		.0143,
		.0123,
		.0095,
		.0078,
		.0063,
		.0048,
		.0039,
		.0032,
	},
	// 1.1
	{
		1.000,
		.6986,
		1.000,
		.8853,
		1.000,
		.8102,
		.7983,
		.6366,
		.5365,
		.4274,
		.3247,
		.2603,
		.2185,
		.1611,
		.1354,
		.1012,
		.0790,
		.0642,
		.0472,
		.0386,
		.0306,
		.0250,
		.0188,
		.0138,
		.0111,
	},
	// 1.2
	{
		1.000,
		.6986,
		1.000,
		.8853,
		.8346,
		.8102,
		.6093,
		.4949,
		.3773,
		.2731,
		.2122,
		.1573,
		.1119,
		.0936,
		.0680,
		.0523,
		.0388,
		.0279,
		.0211,
		.0144,
		.0103,
		.0078,
		.0061,
		.0043,
		.0030,
	},
	// 1.3
	{
		1.000,
		.6986,
		1.000,
		.8853,
		.8346,
		.5752,
		.4433,
		.3721,
		.2510,
		.1859,
		.1310,
		.0999,
		.0727,
		.0509,
		.0422,
		.0275,
		.0192,
		.0131,
		.0094,
		.0066,
	},
	// 1.4
	{
		1.000,
		.6986,
		1.000,
		.8853,
		.6762,
		.4712,
		.3067,
		.2702,
		.1854,
		.1213,
		.0878,
		.0607,
		.0403,
		.0259,
		.0181,
		.0122,
		.0080,
		.0057,
		.0039,
		.0026,
	},
	// 1.5
	{
		1.000,
		.6986,
		1.000,
		.6651,
		.5309,
		.3785,
		.2502,
		.1563,
		.1120,
		.0757,
		.0489,
		.0304,
		.0211,
		.0140,
		.0090,
	},
	// 1.6
	{
		1.000,
		.6986,
		1.000,
		.6651,
		.4034,
		.2980,
		.1599,
		.1036,
		.0774,
		.0452,
		.0303,
		.0166,
		.0104,
		.0072,
		.0048,
	},
	// 1.7
	{
		1.000,
		.6986,
		1.000,
		.6651,
		.4034,
		.2298,
		.1253,
		.0832,
		.0423,
		.0258,
		.0152,
		.0102,
		.0066,
		.0036,
		.0022,
	},
	// 1.8
	{
		1.000,
		.6986,
		.6626,
		.4705,
		.2963,
		.1735,
		.0967,
		.0521,
		.0341,
		.0173,
		.0105,
		.0061,
		.0035,
		.0020,
		.0011,
	},
	// 1.9
	{
		1.000,
		.6986,
		.6626,
		.3124,
		.2101,
		.1283,
		.0737,
		.0406,
		.0217,
		.0114,
	},
	// 2.0
	{
		1.000,
		.6986,
		.6626,
		.3124,
		.2101,
		.0927,
		.0553,
		.0240,
		.0135,
		.0073,
	},
	// 2.1
	{
		1.000,
		.6986,
		.6626,
		.3124,
		.1437,
		.0656,
		.0410,
		.0182,
		.0105,
		.0046,
	},
	// 2.2
	{
		1.000,
		.6986,
		.3828,
		.1940,
		.0947,
		.0656,
		.0299,
		.0136,
		.0062,
		.0037,
	},
	// 2.3
	{
		1.000,
		.6986,
		.3828,
		.1940,
		.0947,
		.0454,
		.0215,
		.0101,
		.0048,
		.0023,
	},
	// 2.4
	{
		1.000,
		.6986,
		.3828,
		.1940,
		.0947,
		.0307,
		.0152,
		.0075,
		.0036,
		.0018,
	},
	// 2.5
	{
		1.000,
		.6986,
		.3828,
		.1124,
		.0602,
		.0307,
		.0152,
		.0075,
		.0027,
		.0014,
	},
	// 2.6
	{
		1.000,
		.6986,
		.1905,
		.1124,
		.0602,
		.0203,
		.0107,
		.0054,
		.0020,
		.0011,
	},
	// 2.7
	{
		1.000,
		.6986,
		.1905,
		.1124,
		.0368,
		.0203,
		.0107,
		.0039,
		.0020,
		.0008,
	},
	// 2.8
	{
		1.000,
		.6986,
		.1905,
		.1124,
		.0368,
		.0131,
		.0073,
		.0039,
		.0015,
		.0008,
	},
	// 2.9
	{
		1.000,
		.6986,
		.1905,
		.0607,
		.0368,
		.0131,
		.0073,
		.0028,
		.0015,
		.0006,
	},
	// 3.0
	{
		1.000,
		.2453,
		.1905,
		.0607,
		.0216,
		.0131,
		.0050,
		.0028,
		.0011,
		.0005,
	},
	// 3.1
	{
		1.000,
		.2453,
		.1905,
		.0607,
		.0216,
		.0083,
		.0050,
		.0020,
		.0008,
		.0005,
	},
	// 3.2
	{
		1.000,
		.2453,
		.0809,
		.0607,
		.0216,
		.0083,
		.0050,
		.0020,
		.0008,
		.0004,
	},
	// 3.3
	{
		1.000,
		.2453,
		.0809,
		.0607,
		.0216,
		.0083,
		.0033,
		.0020,
		.0008,
		.0004,
	},
	// 3.4
	{
		1.000,
		.2453,
		.0809,
		.0304,
		.0216,
		.0083,
		.0033,
		.0014,
		.0006,
		.0004,
	},
	// 3.5
	{
		1.000,
		.2453,
		.0809,
		.0304,
		.0122,
		.0083,
		.0033,
		.0014,
		.0006,
		.0003,
	},
	// 3.6
	{
		1.000,
		.2453,
		.0809,
		.0304,
		.0122,
		.0051,
		.0033,
		.0014,
		.0006,
		.0003,
	},
	// 3.7
	{
		1.000,
		.2453,
		.0809,
		.0304,
		.0122,
		.0051,
		.0022,
		.0014,
		.0006,
		.0003,
	},
	// 3.8
	{
		1.000,
		.2453,
		.0809,
		.0304,
		.0122,
		.0051,
		.0022,
		.0010,
		.0005,
		.0003,
	},
	// 3.9
	{
		1.000,
		.2453,
		.0809,
		.0304,
		.0122,
		.0051,
		.0022,
		.0010,
		.0005,
		.0002,
	},
	// 4.0
	{
		1.000,
		.2453,
		.0809,
		.0304,
		.0122,
		.0051,
		.0022,
		.0010,
		.0005,
		.0002,
	},
}
