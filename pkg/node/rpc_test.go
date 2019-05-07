package node

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/internal/appnet"
	"github.com/skycoin/skywire/pkg/app"
	"github.com/skycoin/skywire/pkg/cipher"
	"github.com/skycoin/skywire/pkg/router"
)

var (
	startChan chan string
	stopChan  chan string
)

func TestListApps(t *testing.T) {
	apps := []AutoStartConfig{
		{App: "foo", Port: 0, Args: []string{}},
		{App: "bar", Port: 3, Args: []string{}},
	}
	metas := map[string]*app.Meta{
		"foo": {AppName: "foo", AppVersion: "1.0", ProtocolVersion: supportedProtocolVersion},
		"bar": {AppName: "bar", AppVersion: "1.0", ProtocolVersion: supportedProtocolVersion},
	}

	rpc := &RPC{&Node{conf: &Config{AutoStartApps: apps}, apps: metas}}

	var reply []*app.Meta
	require.NoError(t, rpc.Apps(nil, &reply))
	require.Len(t, reply, 2)

	// apps are sorted by lexical order of their name
	app1 := reply[0]
	assert.Equal(t, "bar", app1.AppName)
	assert.Equal(t, "1.0", app1.AppVersion)
	assert.Equal(t, supportedProtocolVersion, app1.ProtocolVersion)

	app2 := reply[1]
	assert.Equal(t, "foo", app2.AppName)
	assert.Equal(t, "1.0", app2.AppVersion)
	assert.Equal(t, supportedProtocolVersion, app2.ProtocolVersion)
}

func TestRPC(t *testing.T) {
	startChan = make(chan string)
	stopChan = make(chan string)
	unknownApp := "bar"
	validAppName := "foo"

	pk, sk, err := cipher.GenerateDeterministicKeyPair([]byte("test"))
	require.NoError(t, err)

	node := &Node{
		pm: newMockProcManager(),
		apps: map[string]*app.Meta{
			"foo":  {AppName: "foo", AppVersion: "1.0", ProtocolVersion: supportedProtocolVersion, Host: pk},
			"foo2": {AppName: "foo2", AppVersion: "1.0", ProtocolVersion: supportedProtocolVersion, Host: pk},
		},
		ef: &mockExecutorFactory{},
		conf: &Config{
			Node: KeyFields{
				PubKey: pk,
				SecKey: sk,
			},
		},
		rootBinDir: "apps",
	}

	rpc := &RPC{node: node}

	t.Run("start-stop apps", func(t *testing.T) {
		var pid router.ProcID

		err = rpc.StartProc(&StartProcIn{
			AppName: unknownApp,
			Port:    10,
			Args:    []string{},
		}, &pid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		require.NoError(t, rpc.StartProc(&StartProcIn{
			AppName: validAppName,
			Port:    10,
			Args:    []string{},
		}, &pid))

		mockFactory := node.ef.(*mockExecutorFactory)

		assert.Equal(t, <-startChan, validAppName)
		assert.NotEqual(t, 0, pid)
		assert.Equal(t, "apps/foo", mockFactory.binLoc)
		assert.Equal(t, "foo", mockFactory.workDir)

		mockManager := node.pm.(*mockProcManager)
		require.NoError(t, rpc.StopProc(&mockManager.currentPID, nil))
		assert.Equal(t, <-stopChan, validAppName)

	})

	t.Run("list procs", func(t *testing.T) {
		var pid router.ProcID

		require.NoError(t, rpc.StartProc(&StartProcIn{
			AppName: validAppName,
			Port:    10,
			Args:    []string{},
		}, &pid))

		require.NoError(t, rpc.StartProc(&StartProcIn{
			AppName: validAppName,
			Port:    10,
			Args:    []string{},
		}, &pid))

		ls := make([]*router.ProcInfo, 0)
		require.NoError(t, rpc.ListProcs(nil, &ls))
		assert.Len(t, ls, 2)
	})

	t.Run("apps", func(t *testing.T) {
		apps := make([]*app.Meta, 0)

		require.NoError(t, rpc.Apps(nil, &apps))
		require.Len(t, apps, 2)

		assert.Equal(t, node.apps[apps[0].AppName], apps[0])
		assert.Equal(t, node.apps[apps[1].AppName], apps[1])
	})
}

//
//func TestRPC(t *testing.T) {
//	r := new(mockRouter)
//	executer := new(MockExecuter)
//	defer os.RemoveAll("chat")
//
//	pk1, _, tm1, tm2, errCh, err := transport.MockTransportManagersPair()
//	require.NoError(t, err)
//	defer func() {
//		require.NoError(t, tm1.Close())
//		require.NoError(t, tm2.Close())
//		require.NoError(t, <-errCh)
//		require.NoError(t, <-errCh)
//	}()
//
//	_, err = tm2.CreateTransport(context.TODO(), pk1, "mock", true)
//	require.NoError(t, err)
//
//	conf := &Config{
//		Node: KeyFields{PubKey: pk1},
//		Apps: []AppConfig{
//			{App: "foo", AutoStart: false, Port: 10},
//			{App: "bar", AutoStart: false, Port: 20},
//		},
//	}
//	node := &Node{
//		c:           conf,
//		r:           r,
//		tm:          tm1,
//		rt:          routing.InMemoryRoutingTable(),
//		executer:    executer,
//		startedApps: map[string]*appBind{},
//		logger:      logging.MustGetLogger("test"),
//	}
//
//	require.NoError(t, node.StartApp("foo"))
//	require.NoError(t, node.StartApp("bar"))
//
//	gateway := &RPC{node: node}
//
//	sConn, cConn := net.Pipe()
//	defer func() {
//		require.NoError(t, sConn.Close())
//		require.NoError(t, cConn.Close())
//	}()
//
//	svr := rpc.NewServer()
//	require.NoError(t, svr.RegisterName(RPCPrefix, gateway))
//	go svr.ServeConn(sConn)
//
//	//client := RPCClient{Client: rpc.NewClient(cConn)}
//
//	print := func(t *testing.T, name string, v interface{}) {
//		j, err := json.MarshalIndent(v, name+": ", "  ")
//		require.NoError(t, err)
//		t.log(string(j))
//	}
//
//	t.Run("Summary", func(t *testing.T) {
//		test := func(t *testing.T, summary *Summary) {
//			assert.Equal(t, pk1, summary.PubKey)
//			assert.Len(t, summary.Apps, 2)
//			assert.Len(t, summary.Transports, 1)
//			print(t, "Summary", summary)
//		}
//		t.Run("RPCServer", func(t *testing.T) {
//			var summary Summary
//			require.NoError(t, gateway.Summary(&struct{}{}, &summary))
//			test(t, &summary)
//		})
//		//t.Run("RPCClient", func(t *testing.T) {
//		//	summary, err := client.Summary()
//		//	require.NoError(t, err)
//		//	test(t, summary)
//		//})
//	})
//
//	//t.Run("Apps", func(t *testing.T) {
//	//	test := func(t *testing.T, apps []*AppState) {
//	//		assert.Len(t, apps, 2)
//	//		print(t, "Apps", apps)
//	//	}
//	//	t.Run("RPCServer", func(t *testing.T) {
//	//		var apps []*AppState
//	//		require.NoError(t, gateway.Apps(&struct{}{}, &apps))
//	//		test(t, apps)
//	//	})
//	//	//t.Run("RPCClient", func(t *testing.T) {
//	//	//	apps, err := client.Apps()
//	//	//	require.NoError(t, err)
//	//	//	test(t, apps)
//	//	//})
//	//})
//
//	// TODO(evanlinjin): For some reason, this freezes.
//	//t.Run("StopStartApp", func(t *testing.T) {
//	//	AppName := "foo"
//	//	require.NoError(t, gateway.StopApp(&AppName, &struct{}{}))
//	//	require.NoError(t, gateway.StartApp(&AppName, &struct{}{}))
//	//	require.NoError(t, client.StopApp(AppName))
//	//	require.NoError(t, client.StartApp(AppName))
//	//})
//
//	t.Run("SetAutoStart", func(t *testing.T) {
//		unknownAppName := "whoAmI"
//		AppName := "foo"
//
//		in1 := SetAutoStartIn{AppName: unknownAppName, AutoStart: true}
//		in2 := SetAutoStartIn{AppName: AppName, AutoStart: true}
//		in3 := SetAutoStartIn{AppName: AppName, AutoStart: false}
//
//		// Test with RPC Server
//
//		err := gateway.SetAutoStart(&in1, &struct{}{})
//		require.Error(t, err)
//		assert.Equal(t, ErrUnknownApp, err)
//
//		require.NoError(t, gateway.SetAutoStart(&in2, &struct{}{}))
//		assert.True(t, node.c.Apps[0].AutoStart)
//
//		require.NoError(t, gateway.SetAutoStart(&in3, &struct{}{}))
//		assert.False(t, node.c.Apps[0].AutoStart)
//
//		// Test with RPC Client
//
//		//err = client.SetAutoStart(in1.AppName, in1.AutoStart)
//		//require.Error(t, err)
//		//assert.Equal(t, ErrUnknownApp.Error(), err.Error())
//		//
//		//require.NoError(t, client.SetAutoStart(in2.AppName, in2.AutoStart))
//		//assert.True(t, node.appsConf[0].AutoStart)
//		//
//		//require.NoError(t, client.SetAutoStart(in3.AppName, in3.AutoStart))
//		//assert.False(t, node.appsConf[0].AutoStart)
//	})
//
//	t.Run("TransportTypes", func(t *testing.T) {
//		in := TransportsIn{ShowLogs: true}
//
//		var out []*TransportSummary
//		require.NoError(t, gateway.Transports(&in, &out))
//		assert.Len(t, out, 1)
//		assert.Equal(t, "mock", out[0].Type)
//
//		//out2, err := client.Transports(in.FilterTypes, in.FilterPubKeys, in.ShowLogs)
//		//require.NoError(t, err)
//		//assert.Equal(t, out, out2)
//	})
//
//	t.Run("Transport", func(t *testing.T) {
//		var ids []uuid.UUID
//		node.tm.WalkTransports(func(tp *transport.ManagedTransport) bool {
//			ids = append(ids, tp.ID)
//			return true
//		})
//
//		for _, id := range ids {
//			var summary TransportSummary
//			require.NoError(t, gateway.Transport(&id, &summary))
//
//			//summary2, err := client.Transport(id)
//			//require.NoError(t, err)
//			//require.Equal(t, summary, *summary2)
//		}
//	})
//
//	// TODO: Test add/remove transports
//}

func newMockProcManager() *mockProcManager {
	return &mockProcManager{
		procs:      make(map[router.ProcID]*router.AppProc),
		ports:      make(map[uint16]router.ProcID),
		metas:      make(map[router.ProcID]*app.Meta),
		portOfProc: make(map[router.ProcID]uint16),
		execs:      make(map[router.ProcID]app.Executor),
	}
}

type mockProcManager struct {
	currentPID router.ProcID

	procs      map[router.ProcID]*router.AppProc
	ports      map[uint16]router.ProcID
	portOfProc map[router.ProcID]uint16
	metas      map[router.ProcID]*app.Meta
	execs      map[router.ProcID]app.Executor
}

func (pm *mockProcManager) RunProc(r router.Router, port uint16, exec app.Executor) (*router.AppProc, error) {
	// grab next available pid
	pid := pm.nextFreePID()

	// check Port
	if port != 0 {
		if proc, ok := pm.portAllocated(port); ok {
			return nil, fmt.Errorf("Port already allocated to pid %d", proc.ProcID())
		}
	}

	p, err := router.NewAppProc(pm, r, port, pid, exec)
	if err != nil {
		return nil, err
	}

	pm.portOfProc[pid] = port
	pm.execs[pid] = exec
	pm.procs[pid] = p
	return p, nil
}

func (pm *mockProcManager) AllocPort(pid router.ProcID) uint16 {
	panic("implement me")
}

func (pm *mockProcManager) Proc(pid router.ProcID) (*router.AppProc, bool) {
	p, ok := pm.procs[pid]
	return p, ok
}

func (pm *mockProcManager) ProcOfPort(lPort uint16) (*router.AppProc, bool) {
	panic("implement me")
}

func (pm *mockProcManager) RangeProcIDs(fn router.ProcIDFunc) {
	panic("implement me")
}

func (pm *mockProcManager) RangePorts(fn router.PortFunc) {
	panic("implement me")
}

func (pm *mockProcManager) ListProcs() []*router.ProcInfo {
	procsList := make([]*router.ProcInfo, 0)
	for pid, proc := range pm.procs {
		if !proc.Stopped() {
			procsList = append(procsList, &router.ProcInfo{
				PID:        pid,
				Port:       pm.portOfProc[pid],
				ExecConfig: pm.execs[pid].Config(),
				Meta:       pm.execs[pid].Meta(),
			})
		}
	}

	return procsList
}

func (pm *mockProcManager) Close() error {
	return nil
}

// returns true (with the proc) if given proc of pid is running.
func (pm *mockProcManager) procRunning(pid router.ProcID) (*router.AppProc, bool) {
	if proc, ok := pm.procs[pid]; ok && !proc.Stopped() {
		return proc, true
	}
	return nil, false
}

// returns true (with the proc) id Port is allocated to a running app.
func (pm *mockProcManager) portAllocated(port uint16) (*router.AppProc, bool) {
	pid, ok := pm.ports[port]
	if !ok {
		return nil, false
	}
	return pm.procRunning(pid)
}

// returns the next available and valid pid.
func (pm *mockProcManager) nextFreePID() router.ProcID {
	for {
		if pm.currentPID++; pm.currentPID == 0 {
			continue
		}
		if _, ok := pm.procRunning(pm.currentPID); ok {
			continue
		}
		return pm.currentPID
	}
}

type mockExecutorFactory struct {
	workDir string
	binLoc  string
}

func (m *mockExecutorFactory) New(_ *logging.Logger, meta *app.Meta, c *app.ExecConfig) (app.Executor, error) {
	m.workDir = c.WorkDir
	m.binLoc = c.BinLoc
	return newMockExecutor(meta, c), nil
}

func newMockExecutor(meta *app.Meta, c *app.ExecConfig) *mockExecutor {
	return &mockExecutor{
		stop: make(chan struct{}),
		meta: meta,
		c:    c,
		log:  logging.MustGetLogger("rpc_test"),
	}
}

type mockExecutor struct {
	stop chan struct{}
	meta *app.Meta
	log  *logging.Logger
	c    *app.ExecConfig
}

func (m *mockExecutor) Run(_, _ appnet.HandlerMap) (<-chan struct{}, error) {
	done := make(chan struct{})
	go func() {
		startChan <- m.meta.AppName
		<-m.stop
		done <- struct{}{}
		stopChan <- m.meta.AppName
	}()

	return done, nil
}

func (m *mockExecutor) Stop() error {
	m.stop <- struct{}{}
	return nil
}

func (m *mockExecutor) Config() *app.ExecConfig {
	return m.c
}

func (m *mockExecutor) Meta() *app.Meta {
	return m.meta
}

func (*mockExecutor) Call(t appnet.FrameType, reqData []byte) ([]byte, error) {
	return []byte{}, nil
}

func (*mockExecutor) CallUI(t appnet.FrameType, reqData []byte) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockExecutor) SetLogger(logger *logging.Logger) {
	m.log = logger
}

func (m *mockExecutor) Logger() *logging.Logger {
	return m.log
}
