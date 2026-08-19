package fedratlas

import (
	"context"
	"log"
	"time"

	pb "github.com/fedratlas-org/fedratlas-sync-go/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client handles communication with Fedratlas sync engine
type Client struct {
	grpcClient pb.FedratlasServiceClient
	conn       *grpc.ClientConn
	enabled    bool
}

// Config holds Fedratlas configuration
type Config struct {
	Enabled  bool
	GRPCHost string
	GRPCPort string
	ServerID string
}

// NewClient creates a new Fedratlas client
func NewClient(config Config) (*Client, error) {
	if !config.Enabled {
		log.Println("ℹ️ Fedratlas sync is disabled")
		return &Client{enabled: false}, nil
	}

	address := config.GRPCHost + ":" + config.GRPCPort
	if config.GRPCHost == "" {
		address = "localhost:50051"
	}

	log.Printf("🔗 Connecting to Fedratlas gRPC at %s", address)

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewFedratlasServiceClient(conn)

	return &Client{
		grpcClient: client,
		conn:       conn,
		enabled:    true,
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// OnFeatureChange notifies Fedratlas about a feature change
func (c *Client) OnFeatureChange(featureID int64, activityType string, featureData []byte, version int32, collectionID string) error {
	if !c.enabled {
		return nil // Silently skip if disabled
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.FeatureChangeRequest{
		FeatureId:    featureID,
		ActivityType: activityType,
		FeatureData:  featureData,
		Version:      version,
		CollectionId: collectionID,
	}

	resp, err := c.grpcClient.OnFeatureChange(ctx, req)
	if err != nil {
		log.Printf("⚠️ Fedratlas sync error: %v", err)
		return err
	}

	if !resp.Success {
		log.Printf("⚠️ Fedratlas sync failed: %s", resp.Error)
		return nil
	}

	log.Printf("✅ Fedratlas sync successful for feature %d", featureID)
	return nil
}

// AddPeer adds a new peer server
func (c *Client) AddPeer(serverID, publicKey, endpointURL, status string) error {
	if !c.enabled {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.AddPeerRequest{
		Peer: &pb.Peer{
			ServerId:    serverID,
			PublicKey:   publicKey,
			TrustScore:  0.5,
			EndpointUrl: endpointURL,
			Status:      status,
			LastSeen:    time.Now().Unix(),
			CreatedAt:   time.Now().Unix(),
		},
	}

	resp, err := c.grpcClient.AddPeer(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		log.Printf("⚠️ Failed to add peer: %s", resp.Error)
		return nil
	}

	log.Printf("✅ Peer %s added successfully", serverID)
	return nil
}

// GetPeers returns all connected peers
func (c *Client) GetPeers() ([]*pb.Peer, error) {
	if !c.enabled {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.grpcClient.GetPeers(ctx, &pb.GetPeersRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Peers, nil
}

// GetServerID returns this server's ID
func (c *Client) GetServerID() (string, error) {
	if !c.enabled {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.grpcClient.GetServerID(ctx, &pb.GetServerIDRequest{})
	if err != nil {
		return "", err
	}

	return resp.ServerId, nil
}
