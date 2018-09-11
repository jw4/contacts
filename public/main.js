(function () {
    window.addEventListener('load', function () {
        let addLabel = document.getElementById('addLabel');
        if (addLabel) {
            addLabel.addEventListener('click', function (e) {
                e.preventDefault();
                const labels = document.querySelectorAll('input[name=label]');
                const last = labels[labels.length - 1];
                if (last) {
                    let added = last.cloneNode(false);
                    added.value = '';
                    last.parentElement.insertBefore(added, addLabel);
                }
            });
        }
    });
    console.info('loaded...');
})()
