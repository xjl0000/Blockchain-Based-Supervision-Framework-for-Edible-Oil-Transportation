package blockchain

import (
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// FabricGateway 管理到 Fabric 网络的 Gateway 连接
type FabricGateway struct {
	mu           sync.RWMutex
	org1Contract *client.Contract
	org2Contract *client.Contract
	org1Conn     *grpc.ClientConn
	org2Conn     *grpc.ClientConn
	org1GW       *client.Gateway
	org2GW       *client.Gateway
	enabled      bool
	channel      string
	chaincode    string
}

var gateway *FabricGateway
var once sync.Once

// GetGateway 获取全局 FabricGateway 单例
func GetGateway() *FabricGateway {
	once.Do(func() {
		gateway = &FabricGateway{}
	})
	return gateway
}

// Init 初始化 Fabric Gateway 连接
func Init() error {
	gw := GetGateway()
	gw.enabled = viper.GetBool("fabric.enabled")
	if !gw.enabled {
		log.Println("[Fabric] 区块链功能已禁用，使用模拟模式")
		return nil
	}

	gw.channel = viper.GetString("fabric.channel")
	gw.chaincode = viper.GetString("fabric.chaincode")

	// 初始化 Org1 连接（业务操作身份）
	org1Contract, org1Conn, org1GW, err := connectOrg("fabric.org1")
	if err != nil {
		log.Printf("[Fabric] 警告：Org1 连接失败: %v，将使用离线模式", err)
		gw.enabled = false
		return nil // 不返回错误，允许业务继续
	}
	gw.org1Contract = org1Contract
	gw.org1Conn = org1Conn
	gw.org1GW = org1GW

	// 初始化 Org2 连接（监管操作身份）
	org2Contract, org2Conn, org2GW, err := connectOrg("fabric.org2")
	if err != nil {
		log.Printf("[Fabric] 警告：Org2 连接失败: %v，仅 Org1 可用", err)
		// Org2 失败不阻塞，监管操作可回退到 Org1
	} else {
		gw.org2Contract = org2Contract
		gw.org2Conn = org2Conn
		gw.org2GW = org2GW
	}

	log.Printf("[Fabric] Gateway 连接成功: channel=%s, chaincode=%s", gw.channel, gw.chaincode)
	return nil
}

// connectOrg 根据配置前缀连接指定组织的 Gateway
func connectOrg(configPrefix string) (*client.Contract, *grpc.ClientConn, *client.Gateway, error) {
	mspID := viper.GetString(configPrefix + ".msp_id")
	cryptoPath := viper.GetString(configPrefix + ".crypto_path")
	certRelPath := viper.GetString(configPrefix + ".cert_path")
	keyRelPath := viper.GetString(configPrefix + ".key_path")
	tlsRelPath := viper.GetString(configPrefix + ".tls_cert_path")
	peerEndpoint := viper.GetString(configPrefix + ".peer_endpoint")
	gatewayPeer := viper.GetString(configPrefix + ".gateway_peer")

	certPath := filepath.Join(cryptoPath, certRelPath)
	keyPath := filepath.Join(cryptoPath, keyRelPath)
	tlsCertPath := filepath.Join(cryptoPath, tlsRelPath)

	channelName := viper.GetString("fabric.channel")
	chaincodeName := viper.GetString("fabric.chaincode")

	// 创建 gRPC 连接
	tlsCert, err := loadCertificate(tlsCertPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load TLS cert: %w", err)
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(tlsCert)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	conn, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gRPC dial: %w", err)
	}

	// 创建身份
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("read cert: %w", err)
	}
	certificate, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("parse cert: %w", err)
	}
	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("create identity: %w", err)
	}

	// 创建签名函数
	files, err := os.ReadDir(keyPath)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("read key dir: %w", err)
	}
	if len(files) == 0 {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("no private key found in %s", keyPath)
	}
	keyPEM, err := os.ReadFile(filepath.Join(keyPath, files[0].Name()))
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("read key: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("parse key: %w", err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("create sign: %w", err)
	}

	// 创建 Gateway 连接
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("gateway connect: %w", err)
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	return contract, conn, gw, nil
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certPEM, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read certificate file %s: %w", filename, err)
	}
	return identity.CertificateFromPEM(certPEM)
}

// IsEnabled 返回 Fabric 是否可用
func (gw *FabricGateway) IsEnabled() bool {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	return gw.enabled
}

// GetContract 获取指定组织的合约实例
// role: "监管机构" 使用 Org2，其他使用 Org1
func (gw *FabricGateway) GetContract(role string) *client.Contract {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	if role == "监管机构" && gw.org2Contract != nil {
		return gw.org2Contract
	}
	return gw.org1Contract
}

// Close 关闭所有连接
func (gw *FabricGateway) Close() {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	if gw.org1GW != nil {
		gw.org1GW.Close()
	}
	if gw.org1Conn != nil {
		gw.org1Conn.Close()
	}
	if gw.org2GW != nil {
		gw.org2GW.Close()
	}
	if gw.org2Conn != nil {
		gw.org2Conn.Close()
	}
}
