package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var apiServer string

func main() {
	addr := flag.String("addr", ":3000", "Dashboard address")
	flag.StringVar(&apiServer, "api", "https://soft-justinn-kalvium-66634035.koyeb.app", "API server address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/pods", handlePods)
	mux.HandleFunc("/nodes", handleNodes)
	mux.HandleFunc("/services", handleServices)
	mux.HandleFunc("/deployments", handleDeployments)
	mux.HandleFunc("/namespaces", handleNamespaces)
	mux.HandleFunc("/create", handleCreate)
	mux.HandleFunc("/api/pods", handleAPIPods)
	mux.HandleFunc("/api/nodes", handleAPINodes)
	mux.HandleFunc("/api/services", handleAPIServices)
	mux.HandleFunc("/api/deployments", handleAPIDeployments)
	mux.HandleFunc("/api/namespaces", handleAPINamespaces)
	mux.HandleFunc("/api/stats", handleAPIStats)
	mux.HandleFunc("/api/create/pod", handleCreatePod)
	mux.HandleFunc("/api/create/deployment", handleCreateDeployment)
	mux.HandleFunc("/api/create/service", handleCreateService)
	mux.HandleFunc("/api/create/namespace", handleCreateNamespace)
	mux.HandleFunc("/api/delete/pod", handleDeletePod)
	mux.HandleFunc("/api/delete/deployment", handleDeleteDeployment)
	mux.HandleFunc("/api/delete/service", handleDeleteService)
	mux.HandleFunc("/api/delete/namespace", handleDeleteNamespace)
	mux.HandleFunc("/api/scale", handleScale)

	server := &http.Server{Addr: *addr, Handler: mux}

	go func() {
		fmt.Printf("Dashboard running at http://localhost%s\n", *addr)
		fmt.Printf("API Server: %s\n", apiServer)
		server.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	server.Close()
}

func getStyles() string {
	return `
	<style>
		:root {
			--bg-primary: #0a0e17;
			--bg-secondary: #131a2b;
			--bg-tertiary: #1a2332;
			--bg-card: #161f2e;
			--border-color: #2a3548;
			--text-primary: #ffffff;
			--text-secondary: #8b949e;
			--accent-blue: #58a6ff;
			--accent-green: #3fb950;
			--accent-yellow: #d29922;
			--accent-red: #f85149;
			--accent-purple: #a371f7;
		}
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
			background: var(--bg-primary);
			color: var(--text-primary);
			min-height: 100vh;
			line-height: 1.5;
		}
		.app { display: flex; min-height: 100vh; }

		.sidebar {
			width: 260px;
			background: var(--bg-secondary);
			border-right: 1px solid var(--border-color);
			position: fixed;
			height: 100vh;
			overflow-y: auto;
			z-index: 100;
		}
		.sidebar-header {
			padding: 24px;
			border-bottom: 1px solid var(--border-color);
		}
		.logo {
			display: flex;
			align-items: center;
			gap: 12px;
			font-size: 22px;
			font-weight: 700;
			color: var(--accent-blue);
		}
		.logo-icon {
			width: 40px;
			height: 40px;
			background: linear-gradient(135deg, var(--accent-blue), var(--accent-purple));
			border-radius: 10px;
			display: flex;
			align-items: center;
			justify-content: center;
			font-size: 20px;
		}
		.nav { padding: 16px 0; }
		.nav-section { padding: 8px 24px; font-size: 11px; text-transform: uppercase; color: var(--text-secondary); letter-spacing: 1px; margin-top: 16px; }
		.nav-item {
			display: flex;
			align-items: center;
			gap: 12px;
			padding: 12px 24px;
			color: var(--text-secondary);
			text-decoration: none;
			transition: all 0.2s;
			border-left: 3px solid transparent;
		}
		.nav-item:hover { background: var(--bg-tertiary); color: var(--text-primary); }
		.nav-item.active {
			background: rgba(88, 166, 255, 0.1);
			color: var(--accent-blue);
			border-left-color: var(--accent-blue);
		}
		.nav-icon { font-size: 18px; width: 24px; text-align: center; }

		.main {
			flex: 1;
			margin-left: 260px;
			min-height: 100vh;
		}
		.topbar {
			background: var(--bg-secondary);
			border-bottom: 1px solid var(--border-color);
			padding: 16px 32px;
			display: flex;
			justify-content: space-between;
			align-items: center;
			position: sticky;
			top: 0;
			z-index: 50;
		}
		.topbar-title { font-size: 20px; font-weight: 600; }
		.topbar-actions { display: flex; gap: 12px; align-items: center; }

		.search-box {
			display: flex;
			align-items: center;
			gap: 8px;
			background: var(--bg-tertiary);
			border: 1px solid var(--border-color);
			border-radius: 8px;
			padding: 8px 16px;
			width: 300px;
		}
		.search-box input {
			flex: 1;
			background: none;
			border: none;
			color: var(--text-primary);
			font-size: 14px;
			outline: none;
		}
		.search-box input::placeholder { color: var(--text-secondary); }

		.content { padding: 32px; }

		.stats-grid {
			display: grid;
			grid-template-columns: repeat(4, 1fr);
			gap: 24px;
			margin-bottom: 32px;
		}
		.stat-card {
			background: var(--bg-card);
			border: 1px solid var(--border-color);
			border-radius: 16px;
			padding: 24px;
			position: relative;
			overflow: hidden;
		}
		.stat-card::before {
			content: '';
			position: absolute;
			top: 0;
			left: 0;
			right: 0;
			height: 3px;
		}
		.stat-card.blue::before { background: var(--accent-blue); }
		.stat-card.green::before { background: var(--accent-green); }
		.stat-card.yellow::before { background: var(--accent-yellow); }
		.stat-card.purple::before { background: var(--accent-purple); }
		.stat-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
		.stat-icon { font-size: 24px; opacity: 0.8; }
		.stat-trend { font-size: 12px; padding: 4px 8px; border-radius: 12px; }
		.stat-trend.up { background: rgba(63, 185, 80, 0.2); color: var(--accent-green); }
		.stat-value { font-size: 36px; font-weight: 700; margin-bottom: 4px; }
		.stat-label { color: var(--text-secondary); font-size: 14px; }

		.section { margin-bottom: 32px; }
		.section-header {
			display: flex;
			justify-content: space-between;
			align-items: center;
			margin-bottom: 20px;
		}
		.section-title { font-size: 18px; font-weight: 600; }

		.btn {
			display: inline-flex;
			align-items: center;
			gap: 8px;
			padding: 10px 20px;
			border: none;
			border-radius: 8px;
			font-size: 14px;
			font-weight: 500;
			cursor: pointer;
			transition: all 0.2s;
			text-decoration: none;
		}
		.btn-primary { background: var(--accent-blue); color: white; }
		.btn-primary:hover { background: #4090e0; }
		.btn-secondary { background: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border-color); }
		.btn-secondary:hover { background: var(--bg-card); }
		.btn-danger { background: var(--accent-red); color: white; }
		.btn-danger:hover { background: #d63a3a; }
		.btn-success { background: var(--accent-green); color: white; }
		.btn-success:hover { background: #2ea043; }
		.btn-sm { padding: 6px 12px; font-size: 12px; }
		.btn-icon { padding: 8px; }

		.card {
			background: var(--bg-card);
			border: 1px solid var(--border-color);
			border-radius: 16px;
			overflow: hidden;
		}
		.card-header {
			padding: 20px 24px;
			border-bottom: 1px solid var(--border-color);
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		.card-title { font-weight: 600; }

		.table-wrapper { overflow-x: auto; }
		table { width: 100%; border-collapse: collapse; }
		th {
			text-align: left;
			padding: 14px 24px;
			font-size: 12px;
			font-weight: 600;
			color: var(--text-secondary);
			text-transform: uppercase;
			letter-spacing: 0.5px;
			background: var(--bg-tertiary);
		}
		td {
			padding: 16px 24px;
			border-bottom: 1px solid var(--border-color);
			font-size: 14px;
		}
		tr:last-child td { border-bottom: none; }
		tr:hover td { background: rgba(255,255,255,0.02); }

		.badge {
			display: inline-flex;
			align-items: center;
			gap: 6px;
			padding: 4px 12px;
			border-radius: 20px;
			font-size: 12px;
			font-weight: 500;
		}
		.badge-green { background: rgba(63, 185, 80, 0.15); color: var(--accent-green); }
		.badge-yellow { background: rgba(210, 153, 34, 0.15); color: var(--accent-yellow); }
		.badge-red { background: rgba(248, 81, 73, 0.15); color: var(--accent-red); }
		.badge-blue { background: rgba(88, 166, 255, 0.15); color: var(--accent-blue); }
		.badge-gray { background: rgba(139, 148, 158, 0.15); color: var(--text-secondary); }
		.badge-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }

		.resource-name { font-weight: 500; color: var(--accent-blue); }
		.resource-namespace { color: var(--text-secondary); font-size: 12px; }

		.empty-state {
			text-align: center;
			padding: 60px 20px;
			color: var(--text-secondary);
		}
		.empty-icon { font-size: 48px; margin-bottom: 16px; opacity: 0.5; }
		.empty-text { font-size: 16px; margin-bottom: 8px; }
		.empty-subtext { font-size: 14px; opacity: 0.7; }

		.modal-overlay {
			display: none;
			position: fixed;
			top: 0;
			left: 0;
			right: 0;
			bottom: 0;
			background: rgba(0, 0, 0, 0.8);
			z-index: 200;
			align-items: center;
			justify-content: center;
		}
		.modal-overlay.active { display: flex; }
		.modal {
			background: var(--bg-secondary);
			border: 1px solid var(--border-color);
			border-radius: 16px;
			width: 500px;
			max-height: 90vh;
			overflow-y: auto;
		}
		.modal-header {
			padding: 24px;
			border-bottom: 1px solid var(--border-color);
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		.modal-title { font-size: 18px; font-weight: 600; }
		.modal-close {
			background: none;
			border: none;
			color: var(--text-secondary);
			font-size: 24px;
			cursor: pointer;
			padding: 4px;
		}
		.modal-close:hover { color: var(--text-primary); }
		.modal-body { padding: 24px; }
		.modal-footer {
			padding: 16px 24px;
			border-top: 1px solid var(--border-color);
			display: flex;
			justify-content: flex-end;
			gap: 12px;
		}

		.form-group { margin-bottom: 20px; }
		.form-label { display: block; margin-bottom: 8px; font-size: 14px; font-weight: 500; }
		.form-input {
			width: 100%;
			padding: 12px 16px;
			background: var(--bg-tertiary);
			border: 1px solid var(--border-color);
			border-radius: 8px;
			color: var(--text-primary);
			font-size: 14px;
			transition: border-color 0.2s;
		}
		.form-input:focus { outline: none; border-color: var(--accent-blue); }
		.form-input::placeholder { color: var(--text-secondary); }
		.form-hint { font-size: 12px; color: var(--text-secondary); margin-top: 6px; }

		.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }

		.tabs { display: flex; gap: 4px; margin-bottom: 24px; background: var(--bg-tertiary); padding: 4px; border-radius: 10px; }
		.tab {
			flex: 1;
			padding: 10px 16px;
			border: none;
			background: none;
			color: var(--text-secondary);
			font-size: 14px;
			cursor: pointer;
			border-radius: 8px;
			transition: all 0.2s;
		}
		.tab:hover { color: var(--text-primary); }
		.tab.active { background: var(--accent-blue); color: white; }

		.actions { display: flex; gap: 8px; }

		.toast {
			position: fixed;
			bottom: 24px;
			right: 24px;
			padding: 16px 24px;
			background: var(--bg-secondary);
			border: 1px solid var(--border-color);
			border-radius: 12px;
			display: none;
			align-items: center;
			gap: 12px;
			z-index: 300;
			animation: slideIn 0.3s ease;
		}
		.toast.success { border-left: 4px solid var(--accent-green); }
		.toast.error { border-left: 4px solid var(--accent-red); }
		.toast.show { display: flex; }
		@keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }

		.cluster-info {
			background: var(--bg-card);
			border: 1px solid var(--border-color);
			border-radius: 16px;
			padding: 24px;
			margin-bottom: 32px;
		}
		.cluster-header { display: flex; align-items: center; gap: 16px; margin-bottom: 20px; }
		.cluster-status { display: flex; align-items: center; gap: 8px; }
		.status-dot { width: 10px; height: 10px; border-radius: 50%; background: var(--accent-green); animation: pulse 2s infinite; }
		@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
		.cluster-details { display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px; }
		.cluster-detail-item label { font-size: 12px; color: var(--text-secondary); display: block; margin-bottom: 4px; }
		.cluster-detail-item span { font-weight: 500; }

		.quick-actions { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 32px; }
		.quick-action {
			background: var(--bg-card);
			border: 1px solid var(--border-color);
			border-radius: 12px;
			padding: 20px;
			text-align: center;
			cursor: pointer;
			transition: all 0.2s;
			text-decoration: none;
			color: inherit;
		}
		.quick-action:hover { border-color: var(--accent-blue); transform: translateY(-2px); }
		.quick-action-icon { font-size: 28px; margin-bottom: 12px; }
		.quick-action-title { font-weight: 500; margin-bottom: 4px; }
		.quick-action-desc { font-size: 12px; color: var(--text-secondary); }

		.loading { display: flex; align-items: center; justify-content: center; padding: 40px; }
		.spinner { width: 32px; height: 32px; border: 3px solid var(--border-color); border-top-color: var(--accent-blue); border-radius: 50%; animation: spin 1s linear infinite; }
		@keyframes spin { to { transform: rotate(360deg); } }

		@media (max-width: 1200px) {
			.stats-grid { grid-template-columns: repeat(2, 1fr); }
			.quick-actions { grid-template-columns: repeat(2, 1fr); }
		}
		@media (max-width: 768px) {
			.sidebar { display: none; }
			.main { margin-left: 0; }
			.stats-grid { grid-template-columns: 1fr; }
		}
	</style>`
}

func getScripts() string {
	return `
	<script>
		function showModal(id) {
			document.getElementById(id).classList.add('active');
		}
		function hideModal(id) {
			document.getElementById(id).classList.remove('active');
		}
		function showToast(message, type) {
			const toast = document.getElementById('toast');
			toast.querySelector('.toast-message').textContent = message;
			toast.className = 'toast ' + type + ' show';
			setTimeout(() => toast.classList.remove('show'), 3000);
		}
		async function createPod(e) {
			e.preventDefault();
			const data = {
				name: document.getElementById('pod-name').value,
				namespace: document.getElementById('pod-namespace').value || 'default',
				image: document.getElementById('pod-image').value,
			};
			try {
				const res = await fetch('/api/create/pod', {
					method: 'POST',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify(data)
				});
				if (res.ok) {
					showToast('Pod created successfully', 'success');
					hideModal('create-pod-modal');
					setTimeout(() => location.reload(), 1000);
				} else {
					showToast('Failed to create pod', 'error');
				}
			} catch (err) {
				showToast('Error: ' + err.message, 'error');
			}
		}
		async function createDeployment(e) {
			e.preventDefault();
			const data = {
				name: document.getElementById('deploy-name').value,
				namespace: document.getElementById('deploy-namespace').value || 'default',
				image: document.getElementById('deploy-image').value,
				replicas: parseInt(document.getElementById('deploy-replicas').value) || 1,
			};
			try {
				const res = await fetch('/api/create/deployment', {
					method: 'POST',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify(data)
				});
				if (res.ok) {
					showToast('Deployment created successfully', 'success');
					hideModal('create-deployment-modal');
					setTimeout(() => location.reload(), 1000);
				} else {
					showToast('Failed to create deployment', 'error');
				}
			} catch (err) {
				showToast('Error: ' + err.message, 'error');
			}
		}
		async function createService(e) {
			e.preventDefault();
			const data = {
				name: document.getElementById('svc-name').value,
				namespace: document.getElementById('svc-namespace').value || 'default',
				port: parseInt(document.getElementById('svc-port').value),
				targetPort: parseInt(document.getElementById('svc-target-port').value),
				selector: document.getElementById('svc-selector').value,
			};
			try {
				const res = await fetch('/api/create/service', {
					method: 'POST',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify(data)
				});
				if (res.ok) {
					showToast('Service created successfully', 'success');
					hideModal('create-service-modal');
					setTimeout(() => location.reload(), 1000);
				} else {
					showToast('Failed to create service', 'error');
				}
			} catch (err) {
				showToast('Error: ' + err.message, 'error');
			}
		}
		async function createNamespace(e) {
			e.preventDefault();
			const data = { name: document.getElementById('ns-name').value };
			try {
				const res = await fetch('/api/create/namespace', {
					method: 'POST',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify(data)
				});
				if (res.ok) {
					showToast('Namespace created successfully', 'success');
					hideModal('create-namespace-modal');
					setTimeout(() => location.reload(), 1000);
				} else {
					showToast('Failed to create namespace', 'error');
				}
			} catch (err) {
				showToast('Error: ' + err.message, 'error');
			}
		}
		async function deleteResource(type, namespace, name) {
			if (!confirm('Are you sure you want to delete ' + name + '?')) return;
			try {
				const res = await fetch('/api/delete/' + type + '?namespace=' + namespace + '&name=' + name, { method: 'DELETE' });
				if (res.ok) {
					showToast(name + ' deleted successfully', 'success');
					setTimeout(() => location.reload(), 1000);
				} else {
					showToast('Failed to delete ' + name, 'error');
				}
			} catch (err) {
				showToast('Error: ' + err.message, 'error');
			}
		}
		async function scaleDeployment(namespace, name) {
			const replicas = prompt('Enter number of replicas:');
			if (replicas === null) return;
			try {
				const res = await fetch('/api/scale?namespace=' + namespace + '&name=' + name + '&replicas=' + replicas, { method: 'POST' });
				if (res.ok) {
					showToast('Deployment scaled successfully', 'success');
					setTimeout(() => location.reload(), 1000);
				} else {
					showToast('Failed to scale deployment', 'error');
				}
			} catch (err) {
				showToast('Error: ' + err.message, 'error');
			}
		}
		function filterTable() {
			const filter = document.getElementById('search-input').value.toLowerCase();
			const rows = document.querySelectorAll('tbody tr');
			rows.forEach(row => {
				const text = row.textContent.toLowerCase();
				row.style.display = text.includes(filter) ? '' : 'none';
			});
		}
		setInterval(() => {
			if (document.hidden) return;
			fetch('/api/stats').then(r => r.json()).then(stats => {
				document.querySelectorAll('[data-stat]').forEach(el => {
					const key = el.dataset.stat;
					if (stats[key] !== undefined) el.textContent = stats[key];
				});
			});
		}, 5000);
	</script>`
}

func getSidebar(active string) string {
	navItem := func(href, icon, label, id string) string {
		class := "nav-item"
		if id == active {
			class += " active"
		}
		return `<a href="` + href + `" class="` + class + `"><span class="nav-icon">` + icon + `</span>` + label + `</a>`
	}

	return `
	<aside class="sidebar">
		<div class="sidebar-header">
			<div class="logo">
				<div class="logo-icon">☸</div>
				<span>Kube</span>
			</div>
		</div>
		<nav class="nav">
			<div class="nav-section">Overview</div>
			` + navItem("/", "📊", "Dashboard", "home") + `
			<div class="nav-section">Workloads</div>
			` + navItem("/pods", "📦", "Pods", "pods") + `
			` + navItem("/deployments", "🚀", "Deployments", "deployments") + `
			<div class="nav-section">Network</div>
			` + navItem("/services", "🌐", "Services", "services") + `
			<div class="nav-section">Cluster</div>
			` + navItem("/nodes", "🖥️", "Nodes", "nodes") + `
			` + navItem("/namespaces", "📁", "Namespaces", "namespaces") + `
			<div class="nav-section">Actions</div>
			` + navItem("/create", "➕", "Create Resource", "create") + `
		</nav>
	</aside>`
}

func getToast() string {
	return `<div id="toast" class="toast"><span class="toast-message"></span></div>`
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	stats := getStats()

	html := `<!DOCTYPE html><html><head><title>Dashboard - Kube</title><meta name="viewport" content="width=device-width, initial-scale=1">` + getStyles() + `</head><body>
	<div class="app">` + getSidebar("home") + `
		<main class="main">
			<div class="topbar">
				<div class="topbar-title">Dashboard</div>
				<div class="topbar-actions">
					<div class="cluster-status"><div class="status-dot"></div><span>Cluster Healthy</span></div>
				</div>
			</div>
			<div class="content">
				<div class="cluster-info">
					<div class="cluster-header">
						<h3>Cluster Overview</h3>
					</div>
					<div class="cluster-details">
						<div class="cluster-detail-item">
							<label>API Server</label>
							<span>` + apiServer + `</span>
						</div>
						<div class="cluster-detail-item">
							<label>Status</label>
							<span class="badge badge-green"><span class="badge-dot"></span>Connected</span>
						</div>
						<div class="cluster-detail-item">
							<label>Version</label>
							<span>v1.0.0</span>
						</div>
					</div>
				</div>

				<div class="stats-grid">
					<div class="stat-card blue">
						<div class="stat-header">
							<span class="stat-icon">📦</span>
						</div>
						<div class="stat-value" data-stat="totalPods">` + fmt.Sprintf("%d", stats.TotalPods) + `</div>
						<div class="stat-label">Total Pods</div>
					</div>
					<div class="stat-card green">
						<div class="stat-header">
							<span class="stat-icon">✅</span>
						</div>
						<div class="stat-value" data-stat="runningPods">` + fmt.Sprintf("%d", stats.RunningPods) + `</div>
						<div class="stat-label">Running Pods</div>
					</div>
					<div class="stat-card yellow">
						<div class="stat-header">
							<span class="stat-icon">🖥️</span>
						</div>
						<div class="stat-value" data-stat="totalNodes">` + fmt.Sprintf("%d", stats.TotalNodes) + `</div>
						<div class="stat-label">Nodes</div>
					</div>
					<div class="stat-card purple">
						<div class="stat-header">
							<span class="stat-icon">🌐</span>
						</div>
						<div class="stat-value" data-stat="totalServices">` + fmt.Sprintf("%d", stats.TotalServices) + `</div>
						<div class="stat-label">Services</div>
					</div>
				</div>

				<div class="quick-actions">
					<a href="/create" class="quick-action">
						<div class="quick-action-icon">📦</div>
						<div class="quick-action-title">Create Pod</div>
						<div class="quick-action-desc">Deploy a new pod</div>
					</a>
					<a href="/create" class="quick-action">
						<div class="quick-action-icon">🚀</div>
						<div class="quick-action-title">Create Deployment</div>
						<div class="quick-action-desc">Create a deployment</div>
					</a>
					<a href="/create" class="quick-action">
						<div class="quick-action-icon">🌐</div>
						<div class="quick-action-title">Create Service</div>
						<div class="quick-action-desc">Expose your app</div>
					</a>
					<a href="/create" class="quick-action">
						<div class="quick-action-icon">📁</div>
						<div class="quick-action-title">Create Namespace</div>
						<div class="quick-action-desc">Organize resources</div>
					</a>
				</div>
			</div>
		</main>
	</div>` + getToast() + getScripts() + `</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handlePods(w http.ResponseWriter, r *http.Request) {
	pods := fetchPods()
	rows := ""
	for _, p := range pods {
		badgeClass := getBadgeClass(p.Status)
		rows += `<tr>
			<td><div class="resource-name">` + p.Name + `</div><div class="resource-namespace">` + p.Namespace + `</div></td>
			<td><span class="badge ` + badgeClass + `"><span class="badge-dot"></span>` + p.Status + `</span></td>
			<td>` + p.Ready + `</td>
			<td>` + fmt.Sprintf("%d", p.Restarts) + `</td>
			<td>` + p.Node + `</td>
			<td>` + p.Age + `</td>
			<td class="actions">
				<button onclick="deleteResource('pod', '` + p.Namespace + `', '` + p.Name + `')" class="btn btn-danger btn-sm">Delete</button>
			</td>
		</tr>`
	}
	if len(pods) == 0 {
		rows = `<tr><td colspan="7"><div class="empty-state"><div class="empty-icon">📦</div><div class="empty-text">No pods found</div><div class="empty-subtext">Create your first pod to get started</div></div></td></tr>`
	}

	html := `<!DOCTYPE html><html><head><title>Pods - Kube</title><meta name="viewport" content="width=device-width, initial-scale=1">` + getStyles() + `</head><body>
	<div class="app">` + getSidebar("pods") + `
		<main class="main">
			<div class="topbar">
				<div class="topbar-title">Pods</div>
				<div class="topbar-actions">
					<div class="search-box"><span>🔍</span><input type="text" id="search-input" placeholder="Search pods..." onkeyup="filterTable()"></div>
					<button class="btn btn-primary" onclick="showModal('create-pod-modal')">+ Create Pod</button>
				</div>
			</div>
			<div class="content">
				<div class="card">
					<div class="table-wrapper">
						<table>
							<thead><tr><th>Name</th><th>Status</th><th>Ready</th><th>Restarts</th><th>Node</th><th>Age</th><th>Actions</th></tr></thead>
							<tbody>` + rows + `</tbody>
						</table>
					</div>
				</div>
			</div>
		</main>
	</div>
	<div id="create-pod-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Pod</span><button class="modal-close" onclick="hideModal('create-pod-modal')">×</button></div>
			<form onsubmit="createPod(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="pod-name" class="form-input" placeholder="my-pod" required></div>
					<div class="form-group"><label class="form-label">Namespace</label><input type="text" id="pod-namespace" class="form-input" value="default"></div>
					<div class="form-group"><label class="form-label">Image</label><input type="text" id="pod-image" class="form-input" placeholder="nginx:latest" required><div class="form-hint">Docker image to run</div></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-pod-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>` + getToast() + getScripts() + `</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleNodes(w http.ResponseWriter, r *http.Request) {
	nodes := fetchNodes()
	rows := ""
	for _, n := range nodes {
		badgeClass := getBadgeClass(n.Status)
		rows += `<tr>
			<td><div class="resource-name">` + n.Name + `</div></td>
			<td><span class="badge ` + badgeClass + `"><span class="badge-dot"></span>` + n.Status + `</span></td>
			<td>` + n.CPU + `</td>
			<td>` + n.Memory + `</td>
			<td>` + n.IP + `</td>
			<td>` + n.Age + `</td>
		</tr>`
	}
	if len(nodes) == 0 {
		rows = `<tr><td colspan="6"><div class="empty-state"><div class="empty-icon">🖥️</div><div class="empty-text">No nodes found</div></div></td></tr>`
	}

	html := `<!DOCTYPE html><html><head><title>Nodes - Kube</title><meta name="viewport" content="width=device-width, initial-scale=1">` + getStyles() + `</head><body>
	<div class="app">` + getSidebar("nodes") + `
		<main class="main">
			<div class="topbar">
				<div class="topbar-title">Nodes</div>
				<div class="topbar-actions"><button class="btn btn-secondary" onclick="location.reload()">↻ Refresh</button></div>
			</div>
			<div class="content">
				<div class="card">
					<div class="table-wrapper">
						<table>
							<thead><tr><th>Name</th><th>Status</th><th>CPU</th><th>Memory</th><th>IP Address</th><th>Age</th></tr></thead>
							<tbody>` + rows + `</tbody>
						</table>
					</div>
				</div>
			</div>
		</main>
	</div>` + getToast() + getScripts() + `</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleServices(w http.ResponseWriter, r *http.Request) {
	services := fetchServices()
	rows := ""
	for _, s := range services {
		rows += `<tr>
			<td><div class="resource-name">` + s.Name + `</div><div class="resource-namespace">` + s.Namespace + `</div></td>
			<td><span class="badge badge-blue">` + s.Type + `</span></td>
			<td>` + s.ClusterIP + `</td>
			<td>` + s.Ports + `</td>
			<td>` + s.Age + `</td>
			<td class="actions"><button onclick="deleteResource('service', '` + s.Namespace + `', '` + s.Name + `')" class="btn btn-danger btn-sm">Delete</button></td>
		</tr>`
	}
	if len(services) == 0 {
		rows = `<tr><td colspan="6"><div class="empty-state"><div class="empty-icon">🌐</div><div class="empty-text">No services found</div></div></td></tr>`
	}

	html := `<!DOCTYPE html><html><head><title>Services - Kube</title><meta name="viewport" content="width=device-width, initial-scale=1">` + getStyles() + `</head><body>
	<div class="app">` + getSidebar("services") + `
		<main class="main">
			<div class="topbar">
				<div class="topbar-title">Services</div>
				<div class="topbar-actions"><button class="btn btn-primary" onclick="showModal('create-service-modal')">+ Create Service</button></div>
			</div>
			<div class="content">
				<div class="card">
					<div class="table-wrapper">
						<table>
							<thead><tr><th>Name</th><th>Type</th><th>Cluster IP</th><th>Ports</th><th>Age</th><th>Actions</th></tr></thead>
							<tbody>` + rows + `</tbody>
						</table>
					</div>
				</div>
			</div>
		</main>
	</div>
	<div id="create-service-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Service</span><button class="modal-close" onclick="hideModal('create-service-modal')">×</button></div>
			<form onsubmit="createService(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="svc-name" class="form-input" required></div>
					<div class="form-group"><label class="form-label">Namespace</label><input type="text" id="svc-namespace" class="form-input" value="default"></div>
					<div class="form-row">
						<div class="form-group"><label class="form-label">Port</label><input type="number" id="svc-port" class="form-input" value="80" required></div>
						<div class="form-group"><label class="form-label">Target Port</label><input type="number" id="svc-target-port" class="form-input" value="80" required></div>
					</div>
					<div class="form-group"><label class="form-label">Selector (app=name)</label><input type="text" id="svc-selector" class="form-input" placeholder="app=myapp"></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-service-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>` + getToast() + getScripts() + `</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleDeployments(w http.ResponseWriter, r *http.Request) {
	deployments := fetchDeployments()
	rows := ""
	for _, d := range deployments {
		rows += `<tr>
			<td><div class="resource-name">` + d.Name + `</div><div class="resource-namespace">` + d.Namespace + `</div></td>
			<td>` + d.Ready + `</td>
			<td>` + fmt.Sprintf("%d", d.Available) + `</td>
			<td>` + d.Age + `</td>
			<td class="actions">
				<button onclick="scaleDeployment('` + d.Namespace + `', '` + d.Name + `')" class="btn btn-secondary btn-sm">Scale</button>
				<button onclick="deleteResource('deployment', '` + d.Namespace + `', '` + d.Name + `')" class="btn btn-danger btn-sm">Delete</button>
			</td>
		</tr>`
	}
	if len(deployments) == 0 {
		rows = `<tr><td colspan="5"><div class="empty-state"><div class="empty-icon">🚀</div><div class="empty-text">No deployments found</div></div></td></tr>`
	}

	html := `<!DOCTYPE html><html><head><title>Deployments - Kube</title><meta name="viewport" content="width=device-width, initial-scale=1">` + getStyles() + `</head><body>
	<div class="app">` + getSidebar("deployments") + `
		<main class="main">
			<div class="topbar">
				<div class="topbar-title">Deployments</div>
				<div class="topbar-actions"><button class="btn btn-primary" onclick="showModal('create-deployment-modal')">+ Create Deployment</button></div>
			</div>
			<div class="content">
				<div class="card">
					<div class="table-wrapper">
						<table>
							<thead><tr><th>Name</th><th>Ready</th><th>Available</th><th>Age</th><th>Actions</th></tr></thead>
							<tbody>` + rows + `</tbody>
						</table>
					</div>
				</div>
			</div>
		</main>
	</div>
	<div id="create-deployment-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Deployment</span><button class="modal-close" onclick="hideModal('create-deployment-modal')">×</button></div>
			<form onsubmit="createDeployment(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="deploy-name" class="form-input" required></div>
					<div class="form-group"><label class="form-label">Namespace</label><input type="text" id="deploy-namespace" class="form-input" value="default"></div>
					<div class="form-group"><label class="form-label">Image</label><input type="text" id="deploy-image" class="form-input" placeholder="nginx:latest" required></div>
					<div class="form-group"><label class="form-label">Replicas</label><input type="number" id="deploy-replicas" class="form-input" value="1" min="1"></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-deployment-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>` + getToast() + getScripts() + `</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleNamespaces(w http.ResponseWriter, r *http.Request) {
	namespaces := fetchNamespaces()
	rows := ""
	for _, n := range namespaces {
		badgeClass := getBadgeClass(n.Status)
		rows += `<tr>
			<td><div class="resource-name">` + n.Name + `</div></td>
			<td><span class="badge ` + badgeClass + `"><span class="badge-dot"></span>` + n.Status + `</span></td>
			<td>` + n.Age + `</td>
			<td class="actions"><button onclick="deleteResource('namespace', '', '` + n.Name + `')" class="btn btn-danger btn-sm">Delete</button></td>
		</tr>`
	}
	if len(namespaces) == 0 {
		rows = `<tr><td colspan="4"><div class="empty-state"><div class="empty-icon">📁</div><div class="empty-text">No namespaces found</div></div></td></tr>`
	}

	html := `<!DOCTYPE html><html><head><title>Namespaces - Kube</title><meta name="viewport" content="width=device-width, initial-scale=1">` + getStyles() + `</head><body>
	<div class="app">` + getSidebar("namespaces") + `
		<main class="main">
			<div class="topbar">
				<div class="topbar-title">Namespaces</div>
				<div class="topbar-actions"><button class="btn btn-primary" onclick="showModal('create-namespace-modal')">+ Create Namespace</button></div>
			</div>
			<div class="content">
				<div class="card">
					<div class="table-wrapper">
						<table>
							<thead><tr><th>Name</th><th>Status</th><th>Age</th><th>Actions</th></tr></thead>
							<tbody>` + rows + `</tbody>
						</table>
					</div>
				</div>
			</div>
		</main>
	</div>
	<div id="create-namespace-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Namespace</span><button class="modal-close" onclick="hideModal('create-namespace-modal')">×</button></div>
			<form onsubmit="createNamespace(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="ns-name" class="form-input" required></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-namespace-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>` + getToast() + getScripts() + `</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html><html><head><title>Create Resource - Kube</title><meta name="viewport" content="width=device-width, initial-scale=1">` + getStyles() + `</head><body>
	<div class="app">` + getSidebar("create") + `
		<main class="main">
			<div class="topbar"><div class="topbar-title">Create Resource</div></div>
			<div class="content">
				<div class="quick-actions">
					<div class="quick-action" onclick="showModal('create-pod-modal')">
						<div class="quick-action-icon">📦</div>
						<div class="quick-action-title">Pod</div>
						<div class="quick-action-desc">Run a single container</div>
					</div>
					<div class="quick-action" onclick="showModal('create-deployment-modal')">
						<div class="quick-action-icon">🚀</div>
						<div class="quick-action-title">Deployment</div>
						<div class="quick-action-desc">Manage replicated pods</div>
					</div>
					<div class="quick-action" onclick="showModal('create-service-modal')">
						<div class="quick-action-icon">🌐</div>
						<div class="quick-action-title">Service</div>
						<div class="quick-action-desc">Expose your application</div>
					</div>
					<div class="quick-action" onclick="showModal('create-namespace-modal')">
						<div class="quick-action-icon">📁</div>
						<div class="quick-action-title">Namespace</div>
						<div class="quick-action-desc">Organize resources</div>
					</div>
				</div>
			</div>
		</main>
	</div>
	<div id="create-pod-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Pod</span><button class="modal-close" onclick="hideModal('create-pod-modal')">×</button></div>
			<form onsubmit="createPod(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="pod-name" class="form-input" required></div>
					<div class="form-group"><label class="form-label">Namespace</label><input type="text" id="pod-namespace" class="form-input" value="default"></div>
					<div class="form-group"><label class="form-label">Image</label><input type="text" id="pod-image" class="form-input" placeholder="nginx:latest" required></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-pod-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>
	<div id="create-deployment-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Deployment</span><button class="modal-close" onclick="hideModal('create-deployment-modal')">×</button></div>
			<form onsubmit="createDeployment(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="deploy-name" class="form-input" required></div>
					<div class="form-group"><label class="form-label">Namespace</label><input type="text" id="deploy-namespace" class="form-input" value="default"></div>
					<div class="form-group"><label class="form-label">Image</label><input type="text" id="deploy-image" class="form-input" placeholder="nginx:latest" required></div>
					<div class="form-group"><label class="form-label">Replicas</label><input type="number" id="deploy-replicas" class="form-input" value="1" min="1"></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-deployment-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>
	<div id="create-service-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Service</span><button class="modal-close" onclick="hideModal('create-service-modal')">×</button></div>
			<form onsubmit="createService(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="svc-name" class="form-input" required></div>
					<div class="form-group"><label class="form-label">Namespace</label><input type="text" id="svc-namespace" class="form-input" value="default"></div>
					<div class="form-row">
						<div class="form-group"><label class="form-label">Port</label><input type="number" id="svc-port" class="form-input" value="80"></div>
						<div class="form-group"><label class="form-label">Target Port</label><input type="number" id="svc-target-port" class="form-input" value="80"></div>
					</div>
					<div class="form-group"><label class="form-label">Selector</label><input type="text" id="svc-selector" class="form-input" placeholder="app=myapp"></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-service-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>
	<div id="create-namespace-modal" class="modal-overlay">
		<div class="modal">
			<div class="modal-header"><span class="modal-title">Create Namespace</span><button class="modal-close" onclick="hideModal('create-namespace-modal')">×</button></div>
			<form onsubmit="createNamespace(event)">
				<div class="modal-body">
					<div class="form-group"><label class="form-label">Name</label><input type="text" id="ns-name" class="form-input" required></div>
				</div>
				<div class="modal-footer"><button type="button" class="btn btn-secondary" onclick="hideModal('create-namespace-modal')">Cancel</button><button type="submit" class="btn btn-primary">Create</button></div>
			</form>
		</div>
	</div>` + getToast() + getScripts() + `</body></html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleAPIPods(w http.ResponseWriter, r *http.Request)        { json.NewEncoder(w).Encode(fetchPods()) }
func handleAPINodes(w http.ResponseWriter, r *http.Request)       { json.NewEncoder(w).Encode(fetchNodes()) }
func handleAPIServices(w http.ResponseWriter, r *http.Request)    { json.NewEncoder(w).Encode(fetchServices()) }
func handleAPIDeployments(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(fetchDeployments()) }
func handleAPINamespaces(w http.ResponseWriter, r *http.Request)  { json.NewEncoder(w).Encode(fetchNamespaces()) }
func handleAPIStats(w http.ResponseWriter, r *http.Request)       { json.NewEncoder(w).Encode(getStats()) }

func handleCreatePod(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Namespace, Image string }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Namespace == "" { req.Namespace = "default" }

	pod := map[string]any{"metadata": map[string]string{"name": req.Name, "namespace": req.Namespace}, "spec": map[string]any{"containers": []map[string]string{{"name": req.Name, "image": req.Image}}}}
	data, _ := json.Marshal(pod)
	resp, err := http.Post(apiServer+"/api/v1/namespaces/"+req.Namespace+"/pods", "application/json", strings.NewReader(string(data)))
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
}

func handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Namespace, Image string; Replicas int }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Namespace == "" { req.Namespace = "default" }
	if req.Replicas == 0 { req.Replicas = 1 }

	dep := map[string]any{
		"metadata": map[string]string{"name": req.Name, "namespace": req.Namespace},
		"spec": map[string]any{
			"replicas": req.Replicas,
			"selector": map[string]string{"app": req.Name},
			"template": map[string]any{"containers": []map[string]string{{"name": req.Name, "image": req.Image}}},
		},
	}
	data, _ := json.Marshal(dep)
	resp, err := http.Post(apiServer+"/api/v1/namespaces/"+req.Namespace+"/deployments", "application/json", strings.NewReader(string(data)))
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
}

func handleCreateService(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Namespace, Selector string; Port, TargetPort int }
	json.NewDecoder(r.Body).Decode(&req)
	if req.Namespace == "" { req.Namespace = "default" }

	selector := map[string]string{}
	if req.Selector != "" {
		parts := strings.SplitN(req.Selector, "=", 2)
		if len(parts) == 2 { selector[parts[0]] = parts[1] }
	}

	svc := map[string]any{
		"metadata": map[string]string{"name": req.Name, "namespace": req.Namespace},
		"spec": map[string]any{"ports": []map[string]int{{"port": req.Port, "targetPort": req.TargetPort}}, "selector": selector},
	}
	data, _ := json.Marshal(svc)
	resp, err := http.Post(apiServer+"/api/v1/namespaces/"+req.Namespace+"/services", "application/json", strings.NewReader(string(data)))
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
}

func handleCreateNamespace(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name string }
	json.NewDecoder(r.Body).Decode(&req)

	ns := map[string]any{"metadata": map[string]string{"name": req.Name}}
	data, _ := json.Marshal(ns)
	resp, err := http.Post(apiServer+"/api/v1/namespaces", "application/json", strings.NewReader(string(data)))
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
}

func handleDeletePod(w http.ResponseWriter, r *http.Request) {
	ns, name := r.URL.Query().Get("namespace"), r.URL.Query().Get("name")
	req, _ := http.NewRequest("DELETE", apiServer+"/api/v1/namespaces/"+ns+"/pods/"+name, nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil { defer resp.Body.Close(); w.WriteHeader(resp.StatusCode) }
}

func handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	ns, name := r.URL.Query().Get("namespace"), r.URL.Query().Get("name")
	req, _ := http.NewRequest("DELETE", apiServer+"/api/v1/namespaces/"+ns+"/deployments/"+name, nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil { defer resp.Body.Close(); w.WriteHeader(resp.StatusCode) }
}

func handleDeleteService(w http.ResponseWriter, r *http.Request) {
	ns, name := r.URL.Query().Get("namespace"), r.URL.Query().Get("name")
	req, _ := http.NewRequest("DELETE", apiServer+"/api/v1/namespaces/"+ns+"/services/"+name, nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil { defer resp.Body.Close(); w.WriteHeader(resp.StatusCode) }
}

func handleDeleteNamespace(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	req, _ := http.NewRequest("DELETE", apiServer+"/api/v1/namespaces/"+name, nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil { defer resp.Body.Close(); w.WriteHeader(resp.StatusCode) }
}

func handleScale(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	replicas := r.URL.Query().Get("replicas")

	resp, err := http.Get(apiServer+"/api/v1/namespaces/"+ns+"/deployments/"+name)
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer resp.Body.Close()

	var dep map[string]any
	json.NewDecoder(resp.Body).Decode(&dep)

	if spec, ok := dep["spec"].(map[string]any); ok {
		var r int
		fmt.Sscanf(replicas, "%d", &r)
		spec["replicas"] = r
	}

	data, _ := json.Marshal(dep)
	req, _ := http.NewRequest("PUT", apiServer+"/api/v1/namespaces/"+ns+"/deployments/"+name, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	resp2, _ := http.DefaultClient.Do(req)
	if resp2 != nil { defer resp2.Body.Close(); w.WriteHeader(resp2.StatusCode) }
}

func getBadgeClass(status string) string {
	switch status {
	case "Running", "Ready", "Active": return "badge-green"
	case "Pending", "Unknown": return "badge-yellow"
	case "Failed", "NotReady", "Terminating": return "badge-red"
	default: return "badge-gray"
	}
}

type Stats struct{ TotalPods, RunningPods, TotalNodes, TotalServices, TotalDeployments int }
type Pod struct{ Name, Namespace, Status, Ready, Node, Age string; Restarts int }
type Node struct{ Name, Status, CPU, Memory, IP, Age string }
type Service struct{ Name, Namespace, Type, ClusterIP, Ports, Age string }
type Deployment struct{ Name, Namespace, Ready, Age string; Available int }
type Namespace struct{ Name, Status, Age string }

func getStats() Stats {
	pods, nodes, services, deployments := fetchPods(), fetchNodes(), fetchServices(), fetchDeployments()
	s := Stats{TotalPods: len(pods), TotalNodes: len(nodes), TotalServices: len(services), TotalDeployments: len(deployments)}
	for _, p := range pods { if p.Status == "Running" { s.RunningPods++ } }
	return s
}

func fetchPods() []Pod {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiServer + "/api/v1/pods")
	if err != nil { return nil }
	defer resp.Body.Close()
	var result struct {
		Items []struct {
			Metadata struct{ Name, Namespace, Created string } `json:"metadata"`
			Spec struct{ NodeName string; Containers []struct{ Name string } `json:"containers"` } `json:"spec"`
			Status struct{ Phase string; ContainerStatuses []struct{ Ready bool; RestartCount int } `json:"containerStatuses"` } `json:"status"`
		} `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	var pods []Pod
	for _, i := range result.Items {
		ready, restarts := 0, 0
		for _, cs := range i.Status.ContainerStatuses { if cs.Ready { ready++ }; restarts += cs.RestartCount }
		pods = append(pods, Pod{Name: i.Metadata.Name, Namespace: i.Metadata.Namespace, Status: i.Status.Phase, Ready: fmt.Sprintf("%d/%d", ready, len(i.Spec.Containers)), Restarts: restarts, Node: i.Spec.NodeName, Age: parseAge(i.Metadata.Created)})
	}
	return pods
}

func fetchNodes() []Node {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiServer + "/api/v1/nodes")
	if err != nil { return nil }
	defer resp.Body.Close()
	var result struct {
		Items []struct {
			Metadata struct{ Name, Created string } `json:"metadata"`
			Status struct {
				Conditions []struct{ Type string; Status bool } `json:"conditions"`
				Capacity struct{ CPUCores float64; MemoryMB int64 } `json:"capacity"`
				Addresses []struct{ Type, Address string } `json:"addresses"`
			} `json:"status"`
		} `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	var nodes []Node
	for _, i := range result.Items {
		status, ip := "NotReady", ""
		for _, c := range i.Status.Conditions { if c.Type == "Ready" && c.Status { status = "Ready" } }
		for _, a := range i.Status.Addresses { if a.Type == "InternalIP" { ip = a.Address } }
		nodes = append(nodes, Node{Name: i.Metadata.Name, Status: status, CPU: fmt.Sprintf("%.1f", i.Status.Capacity.CPUCores), Memory: fmt.Sprintf("%dMi", i.Status.Capacity.MemoryMB), IP: ip, Age: parseAge(i.Metadata.Created)})
	}
	return nodes
}

func fetchServices() []Service {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiServer + "/api/v1/namespaces/default/services")
	if err != nil { return nil }
	defer resp.Body.Close()
	var result struct {
		Items []struct {
			Metadata struct{ Name, Namespace, Created string } `json:"metadata"`
			Spec struct{ Type, ClusterIP string; Ports []struct{ Port int; Protocol string } `json:"ports"` } `json:"spec"`
		} `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	var services []Service
	for _, i := range result.Items {
		ports := ""
		for j, p := range i.Spec.Ports { if j > 0 { ports += ", " }; ports += fmt.Sprintf("%d/%s", p.Port, p.Protocol) }
		services = append(services, Service{Name: i.Metadata.Name, Namespace: i.Metadata.Namespace, Type: i.Spec.Type, ClusterIP: i.Spec.ClusterIP, Ports: ports, Age: parseAge(i.Metadata.Created)})
	}
	return services
}

func fetchDeployments() []Deployment {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiServer + "/api/v1/namespaces/default/deployments")
	if err != nil { return nil }
	defer resp.Body.Close()
	var result struct {
		Items []struct {
			Metadata struct{ Name, Namespace, Created string } `json:"metadata"`
			Spec struct{ Replicas int } `json:"spec"`
			Status struct{ ReadyReplicas, AvailableReplicas int } `json:"status"`
		} `json:"items"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	var deployments []Deployment
	for _, i := range result.Items {
		deployments = append(deployments, Deployment{Name: i.Metadata.Name, Namespace: i.Metadata.Namespace, Ready: fmt.Sprintf("%d/%d", i.Status.ReadyReplicas, i.Spec.Replicas), Available: i.Status.AvailableReplicas, Age: parseAge(i.Metadata.Created)})
	}
	return deployments
}

func fetchNamespaces() []Namespace {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiServer + "/api/v1/namespaces")
	if err != nil { return nil }
	defer resp.Body.Close()
	var result struct { Items []struct { Metadata struct{ Name, Created string } `json:"metadata"`; Status struct{ Phase string } `json:"status"` } `json:"items"` }
	json.NewDecoder(resp.Body).Decode(&result)
	var namespaces []Namespace
	for _, i := range result.Items { namespaces = append(namespaces, Namespace{Name: i.Metadata.Name, Status: i.Status.Phase, Age: parseAge(i.Metadata.Created)}) }
	return namespaces
}

func parseAge(created string) string {
	t, err := time.Parse(time.RFC3339, created)
	if err != nil { return "-" }
	d := time.Since(t)
	if d < time.Minute { return fmt.Sprintf("%ds", int(d.Seconds())) }
	if d < time.Hour { return fmt.Sprintf("%dm", int(d.Minutes())) }
	if d < 24*time.Hour { return fmt.Sprintf("%dh", int(d.Hours())) }
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
