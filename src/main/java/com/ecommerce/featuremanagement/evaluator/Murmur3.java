package com.ecommerce.featuremanagement.evaluator;

import java.nio.charset.StandardCharsets;

/** MurmurHash3 x86_32, matching the algorithm used by Go's spaolacci/murmur3 with seed 0. */
final class Murmur3 {

    private static final int C1 = 0xcc9e2d51;
    private static final int C2 = 0x1b873593;

    private Murmur3() {}

    static long sum32(String input) {
        byte[] data = input.getBytes(StandardCharsets.UTF_8);
        int seed = 0;
        int length = data.length;
        int h1 = seed;
        int roundedEnd = length & 0xfffffffc;

        for (int i = 0; i < roundedEnd; i += 4) {
            int k1 = (data[i] & 0xff)
                    | ((data[i + 1] & 0xff) << 8)
                    | ((data[i + 2] & 0xff) << 16)
                    | (data[i + 3] << 24);
            k1 *= C1;
            k1 = Integer.rotateLeft(k1, 15);
            k1 *= C2;
            h1 ^= k1;
            h1 = Integer.rotateLeft(h1, 13);
            h1 = h1 * 5 + 0xe6546b64;
        }

        int k1 = 0;
        switch (length & 0x03) {
            case 3:
                k1 = (data[roundedEnd + 2] & 0xff) << 16;
                // fall through
            case 2:
                k1 |= (data[roundedEnd + 1] & 0xff) << 8;
                // fall through
            case 1:
                k1 |= (data[roundedEnd] & 0xff);
                k1 *= C1;
                k1 = Integer.rotateLeft(k1, 15);
                k1 *= C2;
                h1 ^= k1;
        }

        h1 ^= length;
        h1 ^= (h1 >>> 16);
        h1 *= 0x85ebca6b;
        h1 ^= (h1 >>> 13);
        h1 *= 0xc2b2ae35;
        h1 ^= (h1 >>> 16);

        return h1 & 0xFFFFFFFFL;
    }
}
