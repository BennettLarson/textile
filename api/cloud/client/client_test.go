package client_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	tutil "github.com/textileio/go-threads/util"
	"github.com/textileio/textile/api/apitest"
	c "github.com/textileio/textile/api/cloud/client"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/core"
	"google.golang.org/grpc"
)

func TestClient_Login(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()

	user := apitest.Login(t, client, conf, apitest.NewEmail())
	assert.NotEmpty(t, user.Session)
}

func TestClient_Logout(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()
	ctx := context.Background()

	t.Run("without session", func(t *testing.T) {
		err := client.Logout(ctx)
		require.NotNil(t, err)
	})

	user := apitest.Login(t, client, conf, apitest.NewEmail())

	t.Run("with session", func(t *testing.T) {
		err := client.Logout(common.NewSessionContext(ctx, user.Session))
		require.Nil(t, err)
	})
}

func TestClient_Whoami(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()
	ctx := context.Background()

	t.Run("without session", func(t *testing.T) {
		_, err := client.Whoami(ctx)
		require.NotNil(t, err)
	})

	email := apitest.NewEmail()
	user := apitest.Login(t, client, conf, email)

	t.Run("with session", func(t *testing.T) {
		who, err := client.Whoami(common.NewSessionContext(ctx, user.Session))
		require.Nil(t, err)
		assert.Equal(t, who.ID, user.ID)
		assert.NotEmpty(t, who.Username)
		assert.Equal(t, who.Email, email)
	})
}

func TestClient_GetPrimaryThread(t *testing.T) {
	t.Parallel()
	conf, client, threadsclient, done := setup(t)
	defer done()
	ctx := context.Background()

	user := apitest.Login(t, client, conf, apitest.NewEmail())

	id := thread.NewIDV1(thread.Raw, 32)
	err := threadsclient.NewDB(common.NewSessionContext(ctx, user.Session), id)
	require.Nil(t, err)

	t.Run("without session", func(t *testing.T) {
		_, err := client.GetPrimaryThread(ctx)
		require.NotNil(t, err)
	})

	t.Run("with session", func(t *testing.T) {
		id, err = client.GetPrimaryThread(common.NewSessionContext(ctx, user.Session))
		require.Nil(t, err)
		require.True(t, id.Defined())
	})
}

func TestClient_SetPrimaryThread(t *testing.T) {
	t.Parallel()
	conf, client, threadsclient, done := setup(t)
	defer done()
	ctx := context.Background()

	user := apitest.Login(t, client, conf, apitest.NewEmail())

	id := thread.NewIDV1(thread.Raw, 32)
	err := threadsclient.NewDB(common.NewSessionContext(ctx, user.Session), id)
	require.Nil(t, err)
	ctx = common.NewThreadIDContext(ctx, id)

	t.Run("without session", func(t *testing.T) {
		err := client.SetPrimaryThread(ctx)
		require.NotNil(t, err)
	})

	t.Run("with session", func(t *testing.T) {
		err = client.SetPrimaryThread(common.NewSessionContext(ctx, user.Session))
		require.Nil(t, err)
	})
}

func TestClient_ListThreads(t *testing.T) {
	t.Parallel()
	conf, client, threadsclient, done := setup(t)
	defer done()
	ctx := context.Background()

	t.Run("without session", func(t *testing.T) {
		_, err := client.ListThreads(ctx)
		require.NotNil(t, err)
	})

	user := apitest.Login(t, client, conf, apitest.NewEmail())

	t.Run("with session", func(t *testing.T) {
		ctx = common.NewSessionContext(ctx, user.Session)
		list, err := client.ListThreads(ctx)
		require.Nil(t, err)
		require.Empty(t, list.List)

		err = threadsclient.NewDB(ctx, thread.NewIDV1(thread.Raw, 32))
		require.Nil(t, err)
		err = threadsclient.NewDB(ctx, thread.NewIDV1(thread.Raw, 32))
		require.Nil(t, err)

		list, err = client.ListThreads(ctx)
		require.Nil(t, err)
		require.Equal(t, len(list.List), 2)
	})
}

func TestClient_AddOrg(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()
	ctx := context.Background()

	name := apitest.NewName()

	t.Run("without session", func(t *testing.T) {
		_, err := client.AddOrg(ctx, name)
		require.NotNil(t, err)
	})

	user := apitest.Login(t, client, conf, apitest.NewEmail())

	t.Run("with session", func(t *testing.T) {
		org, err := client.AddOrg(common.NewSessionContext(ctx, user.Session), name)
		require.Nil(t, err)
		assert.NotEmpty(t, org.ID)
		assert.Equal(t, org.Name, name)
	})
}

func TestClient_GetOrg(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()

	name := apitest.NewName()
	user := apitest.Login(t, client, conf, apitest.NewEmail())
	ctx := common.NewSessionContext(context.Background(), user.Session)
	org, err := client.AddOrg(ctx, name)
	require.Nil(t, err)

	t.Run("bad org", func(t *testing.T) {
		_, err := client.GetOrg(common.NewOrgNameContext(ctx, "bad"))
		require.NotNil(t, err)
	})

	t.Run("good org", func(t *testing.T) {
		got, err := client.GetOrg(common.NewOrgNameContext(ctx, org.Name))
		require.Nil(t, err)
		assert.Equal(t, got.ID, org.ID)
	})
}

func TestClient_ListOrgs(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()

	user := apitest.Login(t, client, conf, apitest.NewEmail())
	ctx := common.NewSessionContext(context.Background(), user.Session)

	t.Run("empty", func(t *testing.T) {
		orgs, err := client.ListOrgs(ctx)
		require.Nil(t, err)
		assert.Empty(t, orgs.List)
	})

	name1 := uuid.New().String()
	_, err := client.AddOrg(ctx, name1)
	require.Nil(t, err)
	name2 := uuid.New().String()
	_, err = client.AddOrg(ctx, name2)
	require.Nil(t, err)

	t.Run("not empty", func(t *testing.T) {
		orgs, err := client.ListOrgs(ctx)
		require.Nil(t, err)
		assert.Equal(t, len(orgs.List), 2)
	})
}

func TestClient_RemoveOrg(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()

	name := apitest.NewName()
	user := apitest.Login(t, client, conf, apitest.NewEmail())
	ctx := common.NewSessionContext(context.Background(), user.Session)
	org, err := client.AddOrg(ctx, name)
	require.Nil(t, err)

	t.Run("bad org", func(t *testing.T) {
		err := client.RemoveOrg(common.NewOrgNameContext(ctx, "bad"))
		require.NotNil(t, err)
	})

	user2 := apitest.Login(t, client, conf, apitest.NewEmail())
	ctx2 := common.NewSessionContext(context.Background(), user2.Session)

	t.Run("bad session", func(t *testing.T) {
		err := client.RemoveOrg(common.NewOrgNameContext(ctx2, org.Name))
		require.NotNil(t, err)
	})

	t.Run("good org", func(t *testing.T) {
		octx := common.NewOrgNameContext(ctx, org.Name)
		err := client.RemoveOrg(octx)
		require.Nil(t, err)
		_, err = client.GetOrg(octx)
		require.NotNil(t, err)
	})
}

func TestClient_InviteToOrg(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()

	name := apitest.NewName()
	user := apitest.Login(t, client, conf, apitest.NewEmail())
	ctx := common.NewSessionContext(context.Background(), user.Session)
	org, err := client.AddOrg(ctx, name)
	require.Nil(t, err)
	ctx = common.NewOrgNameContext(ctx, org.Name)

	t.Run("bad email", func(t *testing.T) {
		_, err := client.InviteToOrg(ctx, "jane")
		require.NotNil(t, err)
	})

	t.Run("good email", func(t *testing.T) {
		res, err := client.InviteToOrg(ctx, apitest.NewEmail())
		require.Nil(t, err)
		assert.NotEmpty(t, res.Token)
	})
}

func TestClient_LeaveOrg(t *testing.T) {
	t.Parallel()
	conf, client, _, done := setup(t)
	defer done()

	name := apitest.NewName()
	user := apitest.Login(t, client, conf, apitest.NewEmail())
	ctx := common.NewSessionContext(context.Background(), user.Session)
	org, err := client.AddOrg(ctx, name)
	require.Nil(t, err)
	ctx = common.NewOrgNameContext(ctx, org.Name)

	t.Run("as owner", func(t *testing.T) {
		err := client.LeaveOrg(ctx)
		require.NotNil(t, err)
	})

	user2Email := apitest.NewEmail()
	user2 := apitest.Login(t, client, conf, user2Email)
	ctx2 := common.NewSessionContext(ctx, user2.Session)

	t.Run("as non-member", func(t *testing.T) {
		err := client.LeaveOrg(ctx2)
		require.NotNil(t, err)
	})

	invite, err := client.InviteToOrg(ctx, user2Email)
	require.Nil(t, err)
	_, err = http.Get(fmt.Sprintf("%s/consent/%s", conf.AddrGatewayUrl, invite.Token))
	require.Nil(t, err)

	t.Run("as member", func(t *testing.T) {
		err := client.LeaveOrg(ctx2)
		require.Nil(t, err)
	})
}

func TestClose(t *testing.T) {
	t.Parallel()
	conf, shutdown := apitest.MakeTextile(t)
	defer shutdown()
	target, err := tutil.TCPAddrFromMultiAddr(conf.AddrApi)
	require.Nil(t, err)
	client, err := c.NewClient(target, grpc.WithInsecure(), grpc.WithPerRPCCredentials(common.Credentials{}))
	require.Nil(t, err)

	err = client.Close()
	require.Nil(t, err)
}

func setup(t *testing.T) (core.Config, *c.Client, *tc.Client, func()) {
	conf, shutdown := apitest.MakeTextile(t)
	target, err := tutil.TCPAddrFromMultiAddr(conf.AddrApi)
	require.Nil(t, err)
	opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithPerRPCCredentials(common.Credentials{})}
	client, err := c.NewClient(target, opts...)
	require.Nil(t, err)
	threadsclient, err := tc.NewClient(target, opts...)
	require.Nil(t, err)

	return conf, client, threadsclient, func() {
		shutdown()
		err := client.Close()
		require.Nil(t, err)
		err = threadsclient.Close()
		require.Nil(t, err)
	}
}
