package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

// 接收三个字节切片参数：data（待加密的明文）、key（AES密钥，长度必须是16、24或32字节）、iv（初始化向量，对于AES必须是16字节）
func encryptAES(data, key, iv []byte) (string, error) {
	// 创建密码块：使用传入的 key 创建一个 AES 密码块接口（Block）。如果密钥长度不对，这里会返回错误并退出。
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 分配密文空间：开辟了一块内存 ciphertext。预留出 BlockSize（16字节）的位置用来存放 IV，后面加上明文的长度。
	ciphertext := make([]byte, aes.BlockSize+len(data))
	// 拷贝 IV：创建一个长度为 16 的新切片 ivCopy，并将传入的 iv 拷贝进去。这是为了防止后续加密操作意外修改原始的 iv 变量。
	ivCopy := make([]byte, aes.BlockSize)
	copy(ivCopy, iv)
	// 创建数据流: 使用上一步的密码块和 IV 创建一个 CFB 解密器 (Decrypter)。
	stream := cipher.NewCFBDecrypter(block, ivCopy)
	// 执行异或加密：将明文 data 与数据流进行异或运算，结果存放在 ciphertext 切片从 16 字节开始到最后的位置。
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)
	// 拼接与编码：使用 append 函数将 ivCopy 和实际的密文部分（ciphertext[aes.BlockSize:]）拼接在一起，然后进行 Base64 编码并返回。
	return base64.StdEncoding.EncodeToString(append(ivCopy, ciphertext[aes.BlockSize:]...)), nil
}
