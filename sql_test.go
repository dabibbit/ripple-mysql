package mysql

import (
	"flag"
	"github.com/rubblelabs/ripple/data"
	internal "github.com/rubblelabs/ripple/testing"
	. "launchpad.net/gocheck"
	"testing"
)

var connectionstring = flag.String("connection_string", "ripple:ripple123@/rippletest", "connection string to run tests against")

func init() {
	flag.Parse()
}

func Test(t *testing.T) { TestingT(t) }

type SqlSuite struct{}

var _ = Suite(&SqlSuite{})

func (s *SqlSuite) TestMySql(c *C) {
	db, err := NewMySqlDB(*connectionstring, true)
	c.Assert(err, IsNil)
	var hashes []data.Hash256
	for _, test := range internal.Nodes {
		nodeId, err := data.NewHash256(test.NodeId())
		c.Assert(err, IsNil)
		node, err := data.ReadPrefix(test.Reader(), *nodeId)
		c.Assert(err, IsNil)
		c.Assert(node, NotNil)
		switch node.(type) {
		case *data.TransactionWithMetaData, *data.Ledger:
			hashes = append(hashes, *node.GetHash())
			c.Assert(db.Insert(node), IsNil, Commentf(test.Description))
		}
	}
	items, err := db.GetLookups("GetAccounts")
	c.Assert(err, IsNil)
	c.Assert(len(items), Equals, 47)
	c.Assert(db.GetAccount(0), NotNil)
	c.Assert(db.GetAccount(100), IsNil)
	for _, hash := range hashes {
		node, err := db.Get(hash)
		c.Assert(err, IsNil, Commentf(hash.String()))
		c.Assert(node, NotNil)
		c.Assert(node.GetHash().String(), Equals, hash.String())
		// spew.Printf("%#+v\n", node)
	}
	accounts, err := db.SearchAccounts("r")
	c.Assert(err, IsNil)
	c.Assert(len(accounts), Not(Equals), 0)
}
