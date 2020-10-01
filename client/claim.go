package db

type Claim struct {
	client *Client
	stream Stream
	service string
}

func (cl *Claim) Update(Vars interface{}) {
	_, err := client.db.Put(ctx, "stream:s1:transcode", "bar", clientv3.WithLease(client.lease))
	if err != nil {
		log.Fatal(err)
	}
}