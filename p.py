import hashlib, zlib
def hash_object(data, obj_type, write=True):
    """Compute hash of object data of given type and write to object store if
    "write" is True. Return SHA-1 object hash as hex string.
    """
    header = '{} {}'.format(obj_type, len(data)).encode()
    full_data = header + b'\x00' + data
    sha1 = hashlib.sha1(full_data).hexdigest()
    return sha1

print(zlib.compress(b"Hello World").hex())
print(zlib.decompress(zlib.compress(b"Hello World")))