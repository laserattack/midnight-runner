class ApiClient {
    static sendJSON(jsonData, endpoint) {
        return fetch(endpoint, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(jsonData)
        }).then(response => {
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            return response;
        });
    }

    static receiveJSON(endpoint) {
        return fetch(endpoint)
            .then(response => {
                if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
                return response.json();
            });
    }
}

class DateFormatter {
    static format(timestamp) {
        if (!timestamp) return 'Unknown';
        const date = new Date(timestamp * 1000);
        const pad = (n) => n.toString().padStart(2, '0');
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
    }
}

class Modal {
    constructor(modalId) {
        this.modal = document.getElementById(modalId);
    }

    open() {
        this.modal.style.display = 'block';
    }

    close() {
        this.modal.style.display = 'none';
    }

    static attachEscapeKey(modalInstance) {
        document.addEventListener('keydown', function(event) {
            if (event.key === 'Escape') {
                modalInstance.close();
            }
        });
    }
}

class LogsModal extends Modal {
    constructor() {
        super('logsModal');
        this.refreshInterval = null;
        this.filterInput = null;
        this.allLogs = [];
    }

    open() {
        super.open();
        this.createFilter();
        this.loadLogs();

        setTimeout(() => {
            const logsContent = document.getElementById('logsContent');
            if (logsContent) {
                logsContent.scrollTop = logsContent.scrollHeight;
            }
        }, 0);

        this.refreshInterval = setInterval(() => this.loadLogs(), 1000);
    }

    close() {
        super.close();
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }

    createFilter() {
        if (document.getElementById('logsFilter')) return;

        const modalContent = this.modal.querySelector('.modal-content');
        const filterContainer = document.createElement('div');
        filterContainer.style.marginBottom = '10px';

        this.filterInput = document.createElement('input');
        this.filterInput.type = 'text';
        this.filterInput.id = 'logsFilter';
        this.filterInput.placeholder = 'Filter logs...';
        this.filterInput.className = 'log-filter-input';

        filterContainer.appendChild(this.filterInput);
        const logsContent = this.modal.querySelector('#logsContent');
        modalContent.insertBefore(filterContainer, logsContent);

        this.filterInput.addEventListener('input', () => this.updateLogsDisplay());
    }

    loadLogs() {
        const logsContent = document.getElementById('logsContent');
        if (!window.getSelection().isCollapsed) return;

        const wasAtBottom = logsContent.scrollHeight - logsContent.scrollTop <= logsContent.clientHeight + 5;

ApiClient.sendJSON({ count: 100 }, "/api/last_log")
    .then(response => response.json())
    .then(entries => {
        if (entries.length === 0) {
            this.allLogs = [];
            this.updateLogsDisplay();
            return;
        }

        this.allLogs = entries.map(entry => {
            let attrsText = '';
            if (entry.attrs) {
                const rawAttrs = JSON.stringify(entry.attrs, null, 0);
                if (rawAttrs.length > 1000) {
                    attrsText = ' {too big! see log file}';
                } else {
                    attrsText = ' ' + rawAttrs;
                }
            }

            return {
                displayText: `${entry.time || ''} — ${entry.level || ''} — ${entry.message || ''}${attrsText}`,
                searchContent: `${entry.time || ''} — ${entry.level || ''} — ${entry.message || ''}${attrsText}`.toLowerCase()
            };
        });

        this.updateLogsDisplay();

        if (wasAtBottom) logsContent.scrollTop = logsContent.scrollHeight;
    })
    .catch(err => {
        console.error("Failed to load logs:", err);
        logsContent.textContent = 'Error loading logs';
    });
    }

    updateLogsDisplay() {
        const logsContent = document.getElementById('logsContent');
        const searchTerm = this.filterInput?.value.toLowerCase() || '';
        const filteredLogs = this.allLogs.filter(log => log.searchContent.includes(searchTerm));
        logsContent.textContent = filteredLogs.length
            ? filteredLogs.map(log => log.displayText).join('\n\n')
            : (searchTerm ? 'No matching logs found' : 'No logs available');
    }
}

class JobsTable {
    constructor() {
        this.refreshInterval = null;
    }

    update(data) {
        if (!window.getSelection().isCollapsed) return;

        const jobs = data.jobs || {};
        const jobsArray = Object.entries(jobs);

        let enabled = 0, disabled = 0, active = 0;
        jobsArray.forEach(([_, job]) => {
            if (job.config.status === 'E') enabled++;
            else if (job.config.status === 'D') disabled++;
            else if (job.config.status === "AE") active++;
            else if (job.config.status === "AD") active++;
        });

        document.getElementById('totalJobs').textContent = jobsArray.length;
        document.getElementById('enabledJobs').textContent = enabled;
        document.getElementById('disabledJobs').textContent = disabled;
        document.getElementById('activeJobs').textContent = active;

        const tbody = document.getElementById('jobsTableBody');
        tbody.innerHTML = '';

        if (jobsArray.length === 0) {
            this.showError('No jobs configured');
            return;
        }

        this.showContent();

        jobsArray.forEach(([name, job]) => {
            const row = document.createElement('tr');
            const statusHTML = this.getStatusHTML(job.config.status);
            row.innerHTML = `
                <td class="job-name">${name}</td>
                <td>${job.type}</td>
                <td>${job.description}</td>
                <td><code>${job.config.command}</code></td>
                <td><code>${job.config.cron_expression}</code></td>
                <td>${statusHTML}</td>
                <td>${job.config.timeout}</td>
                <td>${job.config.max_retries}</td>
                <td>${job.config.retry_interval}</td>
            `;
            tbody.appendChild(row);
        });
    }

    getStatusHTML(status) {
        switch(status) {
            case "D": return `<span style="color: #939393;"><b>${status}</b></span>`;
            case "E": return `<span style="color: #7CB342;"><b>${status}</b></span>`;
            default: return `<span style="color: #FFCA29"><b>${status}</b></span>`;
        }
    }

    showError(message) {
        document.getElementById('error').style.display = 'block';
        document.getElementById('error').innerHTML = `<p>${message}</p>`;
        document.getElementById('content').style.display = 'none';
    }

    showContent() {
        document.getElementById('content').style.display = 'block';
        document.getElementById('error').style.display = 'none';
    }

    startAutoRefresh() {
        const refresh = () => {
            ApiClient.receiveJSON("/api/get_database")
                .then(data => {
                    if (!window.getSelection().isCollapsed) return;
                    this.update(data);
                    document.getElementById('lastUpdate').textContent = `Last edit: ${DateFormatter.format(data.metadata?.updated_at)}`;
                    this.showContent();
                })
                .catch(error => {
                    console.error('Error loading database:', error);
                    this.showError('Failed to load jobs data');
                });
        };

        refresh();
        this.refreshInterval = setInterval(refresh, 1000);
    }

    stopAutoRefresh() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }
}

class ManageJobModal extends Modal {
    constructor() {
        super('manageJobModal');
    }

    getJobName() {
        const name = document.getElementById('manageJobName').value.trim();
        if (!name) {
            throw new Error("Job name is empty");
        }
        return name;
    }

    deleteJob() {
        try {
            const name = this.getJobName();
            ApiClient.sendJSON({ name }, "/api/delete_job")
                .then(() => this.close())
                .catch(err => {
                    console.error("Failed to delete job:", err);
                });
        } catch (e) {}
    }

    execJob() {
        try {
            const name = this.getJobName();
            ApiClient.sendJSON({ name }, "/api/exec_job")
                .then(() => this.close())
                .catch(err => {
                    console.error("Failed to exec job:", err);
                });
        } catch (e) {}
    }

    toggleJob() {
        try {
            const name = this.getJobName();
            ApiClient.sendJSON({ name }, "/api/toggle_job")
                .then(() => this.close())
                .catch(err => {
                    console.error("Failed to toggle job:", err);
                });
        } catch (e) {}
    }
}

class SetJobModal extends Modal {
    constructor() {
        super('setJobModal');
    }

    updateCronDescription(cronExpression) {
        const descriptionElement = document.getElementById('cronDescription');
        
        if (!cronExpression || cronExpression.trim() === '') {
            descriptionElement.textContent = '';
            return;
        }

        const parts = cronExpression.trim().split(/\s+/);
        if (parts.length !== 6) {
            descriptionElement.textContent = '(Invalid cron expression)';
            return;
        }

        try {
            const description = cronstrue.toString(cronExpression, {
                throwExceptionOnParseError: true
            });
            descriptionElement.textContent = `(${description})`;
        } catch (error) {
            descriptionElement.textContent = '(Invalid cron expression)';
        }
    }

    open() {
        super.open();
        document.getElementById('setJobForm').reset();
        
        const descriptionElement = document.getElementById('cronDescription');
        descriptionElement.textContent = '';
        
        const cronInput = document.querySelector('input[name="cron"]');
        
        cronInput.addEventListener('input', (e) => {
            this.updateCronDescription(e.target.value);
        });
        
        this.updateCronDescription(cronInput.value);
    }

    attachSubmitHandler() {
        document.getElementById('setJobForm').addEventListener('submit', (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const jobData = {
                name: formData.get('name'),
                description: formData.get('description'),
                command: formData.get('command'),
                cron: formData.get('cron'),
                timeout: parseInt(formData.get('timeout')),
                maxRetries: parseInt(formData.get('maxRetries')),
                retryInterval: parseInt(formData.get('retryInterval'))
            };

            ApiClient.sendJSON(jobData, "/api/change_job")
                .then(() => this.close())
                .catch(err => {
                    console.error("Failed to save job:", err);
                });
        });
    }
}

class App {
    constructor() {
        this.jobsTable = new JobsTable();
        this.logsModal = new LogsModal();
        this.manageJobModal = new ManageJobModal();
        this.setJobModal = new SetJobModal();

        this.setJobModal.attachSubmitHandler();
        this.attachGlobalEventListeners();
    }

    attachGlobalEventListeners() {
        document.addEventListener('DOMContentLoaded', () => this.jobsTable.startAutoRefresh());

        document.addEventListener('visibilitychange', () => {
            if (document.hidden) this.jobsTable.stopAutoRefresh();
            else this.jobsTable.startAutoRefresh();
        });

        document.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                if (this.logsModal.modal.style.display === 'block') this.logsModal.close();
                else if (this.setJobModal.modal.style.display === 'block') this.setJobModal.close();
                else if (this.manageJobModal.modal.style.display === 'block') this.manageJobModal.close();
            }
        });
    }
}

const app = new App();
