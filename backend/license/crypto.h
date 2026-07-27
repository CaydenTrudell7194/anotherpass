#ifndef LICENSE_CRYPTO_H
#define LICENSE_CRYPTO_H

#ifdef __cplusplus
extern "C" {
#endif

/* 采集机器指纹: 返回 SHA256 hex 字符串, 调用者需 free() */
char* get_machine_fingerprint(void);

/* RSA 公钥验签: data 原始数据, sig_b64 base64 签名, pubkey_pem PEM 公钥
 * 返回 1=验证通过, 0=失败 */
int verify_rsa_signature(const char* data, const char* sig_b64, const char* pubkey_pem);

/* XOR 解码字符串 (原地操作) */
void xor_decode(char* str, const char* key, int key_len);

#ifdef __cplusplus
}
#endif

#endif
