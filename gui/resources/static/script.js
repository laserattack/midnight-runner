function sendJSON(jsonData, endpoint) {
    return fetch(endpoint, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(jsonData)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
    });
}

function receiveJSON(endpoint) {
    return fetch(endpoint)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        });
}

function formatTimestamp(timestamp) {
    if (!timestamp) return 'Unknown';
    const date = new Date(timestamp * 1000);

    const pad = (n) => n.toString().padStart(2, '0');

    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1);
    const day = pad(date.getDate());
    const hours = pad(date.getHours());
    const minutes = pad(date.getMinutes());
    const seconds = pad(date.getSeconds());

    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
}

function updateJobsTable(data) {
    const jobs = data.jobs || {};
    const jobsArray = Object.entries(jobs);

    let enabledCount = 0;
    let disabledCount = 0;

    jobsArray.forEach(([name, job]) => {
        if (job.config.status === '💚') {
            enabledCount++;
        } else if (job.config.status === '🩶') {
            disabledCount++;
        }
    });

    document.getElementById('totalJobs')
        .textContent = jobsArray.length;

    document.getElementById('enabledJobs')
        .textContent = enabledCount;

    document.getElementById('disabledJobs')
        .textContent = disabledCount;

    const tableBody = document.getElementById('jobsTableBody');
    tableBody.innerHTML = '';

    if (jobsArray.length === 0) {
        document.getElementById('error').style.display = 'block';
        document.getElementById('error').innerHTML =
            '<p>No jobs configured</p>';
        document.getElementById('content').style.display = 'none';
        return;
    }

    jobsArray.forEach(([name, job]) => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td class="job-name">${name}</td>
            <td>${job.type}</td>
            <td>${job.description}</td>
            <td><code>${job.config.command}</code></td>
            <td><code>${job.config.cron_expression}</code></td>
            <td>${job.config.status}</td>
            <td>${job.config.timeout}</td>
            <td>${job.config.max_retries}</td>
            <td>${job.config.retry_interval}</td>
        `;
        tableBody.appendChild(row);
    });

    document.getElementById('content').style.display = 'block';
    document.getElementById('error').style.display = 'none';
}

