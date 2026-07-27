#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <openssl/evp.h>
#include <openssl/pem.h>
#include <openssl/sha.h>
#include <openssl/bio.h>

#ifdef _WIN32
#include <windows.h>
#include <iphlpapi.h>
#pragma comment(lib, "iphlpapi.lib")
#pragma comment(lib, "advapi32.lib")
#else
#include <unistd.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <net/if.h>
#include <netinet/in.h>
#include <sys/utsname.h>
#endif

/* 机器指纹采集 */
char* get_machine_fingerprint(void) {
    unsigned char hash[SHA256_DIGEST_LENGTH];
    SHA256_CTX sha256;
    SHA256_Init(&sha256);

#ifdef _WIN32
    /* Windows: 采集 主板序列号 + MAC 地址 */
    HKEY hKey;
    char board_serial[256] = {0};
    DWORD buf_size = sizeof(board_serial);
    if (RegOpenKeyExA(HKEY_LOCAL_MACHINE,
        "HARDWARE\\DESCRIPTION\\System\\BIOS", 0, KEY_READ, &hKey) == ERROR_SUCCESS) {
        RegQueryValueExA(hKey, "BaseBoardSerialNumber", NULL, NULL,
            (LPBYTE)board_serial, &buf_size);
        RegCloseKey(hKey);
    }
    SHA256_Update(&sha256, board_serial, strlen(board_serial));

    /* MAC 地址 */
    IP_ADAPTER_INFO adapter_info[16];
    DWORD adapter_size = sizeof(adapter_info);
    if (GetAdaptersInfo(adapter_info, &adapter_size) == NO_ERROR) {
        PIP_ADAPTER_INFO p = adapter_info;
        while (p) {
            SHA256_Update(&sha256, p->Address, p->AddressLength);
            p = p->Next;
        }
    }
#else
    /* Linux: 采集 机器ID + MAC 地址 + hostname */
    FILE *f = fopen("/etc/machine-id", "r");
    if (f) {
        char buf[64] = {0};
        if (fgets(buf, sizeof(buf), f)) {
            SHA256_Update(&sha256, buf, strlen(buf));
        }
        fclose(f);
    }

    /* hostname */
    struct utsname un;
    if (uname(&un) == 0) {
        SHA256_Update(&sha256, un.nodename, strlen(un.nodename));
    }

    /* MAC 地址 (取第一个非 lo 接口) */
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock >= 0) {
        struct ifconf ifc;
        char ifc_buf[4096];
        ifc.ifc_len = sizeof(ifc_buf);
        ifc.ifc_buf = ifc_buf;
        if (ioctl(sock, SIOCGIFCONF, &ifc) == 0) {
            struct ifreq* it = ifc.ifc_req;
            int count = ifc.ifc_len / sizeof(struct ifreq);
            for (int i = 0; i < count; i++) {
                if (strcmp(it[i].ifr_name, "lo") != 0) {
                    if (ioctl(sock, SIOCGIFHWADDR, &it[i]) == 0) {
                        SHA256_Update(&sha256, it[i].ifr_hwaddr.sa_data, 6);
                        break;
                    }
                }
            }
        }
        close(sock);
    }
#endif

    SHA256_Final(hash, &sha256);

    /* 转 hex 字符串 */
    char* result = (char*)malloc(65);
    if (!result) return NULL;
    for (int i = 0; i < 32; i++) {
        sprintf(result + i * 2, "%02x", hash[i]);
    }
    result[64] = '\0';
    return result;
}

/* RSA 公钥验签 */
int verify_rsa_signature(const char* data, const char* sig_b64, const char* pubkey_pem) {
    if (!data || !sig_b64 || !pubkey_pem) return 0;

    BIO* bio = BIO_new_mem_buf(pubkey_pem, -1);
    if (!bio) return 0;

    EVP_PKEY* pkey = PEM_read_bio_PUBKEY(bio, NULL, NULL, NULL);
    BIO_free(bio);
    if (!pkey) return 0;

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) { EVP_PKEY_free(pkey); return 0; }

    /* base64 解码签名（单行格式，无换行） */
    BIO* b64 = BIO_new(BIO_f_base64());
    BIO_set_flags(b64, BIO_FLAGS_BASE64_NO_NL);
    BIO* bmem = BIO_new_mem_buf(sig_b64, -1);
    BIO_push(b64, bmem);

    unsigned char sig[512];
    int sig_len = BIO_read(b64, sig, sizeof(sig));
    BIO_free_all(b64);

    if (sig_len <= 0) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pkey);
        return 0;
    }

    int result = 0;
    if (EVP_DigestVerifyInit(ctx, NULL, EVP_sha256(), NULL, pkey) == 1) {
        EVP_DigestVerifyUpdate(ctx, data, strlen(data));
        result = (EVP_DigestVerifyFinal(ctx, sig, sig_len) == 1);
    }

    EVP_MD_CTX_free(ctx);
    EVP_PKEY_free(pkey);
    return result;
}

/* XOR 解码 */
void xor_decode(char* str, const char* key, int key_len) {
    if (!str || !key || key_len <= 0) return;
    int len = strlen(str);
    for (int i = 0; i < len; i++) {
        str[i] ^= key[i % key_len];
    }
}
