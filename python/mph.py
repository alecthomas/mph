import functools
import itertools

import pyhash


from chd_pb2 import CHDProto


# FNV1a with the same seed as Go.
chd_hash = functools.partial(pyhash.fnv1a_64(), seed=14695981039346656037)


class CHD(object):
    def __init__(self, filename):
        with open(filename) as fd:
            pb = CHDProto()
            pb.ParseFromString(fd.read())
            self._r = pb.r
            self._indices = pb.indices
            self._keys = pb.keys
            self._values = pb.values

    def __getitem__(self, key):
        r0 = self._r[0]
        h = chd_hash(key) ^ r0
        i = h % len(self._indices)
        ri = self._indices[i]
        r = self._r[ri]
        ti = (h ^ r) % len(self._keys)
        if self._keys[ti] != key:
            raise KeyError(key, self._keys[ti])
        return self._values[ti]

    def __contains__(self, key):
        try:
            self[key]
            return True
        except KeyError:
            return False

    def iterkeys(self):
        return iter(self._keys)

    def keys(self):
        return self._keys

    def itervalues(self):
        return iter(self._values)

    def values(self):
        return self._values

    def __iter__(self):
        return iter(self._keys)

    def iteritems(self):
        return iter(itertools.izip(self._keys, self._values))

    def __len__(self):
        return len(self._keys)
