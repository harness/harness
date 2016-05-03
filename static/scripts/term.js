var Filter, defaults, __slice = [].slice;

defaults = {
    useClasses: true
};

extend = function() {
    var dest, k, obj, objs, v, _j, _len;
    dest = arguments[0], objs = 2 <= arguments.length ? __slice.call(arguments, 1) : [];
    for (_j = 0, _len = objs.length; _j < _len; _j++) {
        obj = objs[_j];
        for (k in obj) {
            v = obj[k];
            dest[k] = v;
        }
    }
    return dest;
};

Filter = (function() {
    function Filter(options) {
        if (options === null) {
            options = {};
        }
        this.opts = extend({}, defaults, options);
    }

    Filter.prototype.append = function(element, text) {
        var lines = text.match(/(|.+)\n/g);
        var scope = this;

        lines.forEach(function(line) {

            if (scope.createsFold(line) || !scope.getCurrentFold(element)) {
                scope.appendFold(element);
            }

            if (scope.removesPreviousLine(line)) {
                element.removeChild(scope.getPreviousLine());
            }

            line = scope.createLine(line);
            scope.getCurrentFold(element).appendChild(line);

            return;
        });
    };

    Filter.prototype.createLine = function(line) {
        line = ansi_up.ansi_to_html(line, { 'use_classes': this.opts.useClasses });

        if (line === '') {
            line = '&nbsp;';
        }

        var element = document.createElement('p');
        element.className = 'line';
        element.innerHTML = line;

        return element;
    };

    Filter.prototype.appendFold = function(element) {
        var fold = document.createElement('div');
        fold.className = 'fold';

        var handle = document.createElement('a');
        handle.className = 'fold-handle';
        handle.addEventListener('click', function(event) {
            event.target.parentNode.classList.toggle('open');
        });

        fold.appendChild(handle);
        element.appendChild(fold);
    };

    Filter.prototype.getCurrentFold = function(element) {
        return element.querySelector('div.fold:last-child');
    };

    Filter.prototype.getPreviousLine = function(element) {
        return this.getCurrentFold().querySelector('p.line:last-child');
    };

    Filter.prototype.createsFold = function(line) {
        return (line.substring(0,2) === '$ ' || line.substring(0,7) === '[info] ');
    };

    Filter.prototype.removesPreviousLine = function(line) {
        return (line.substring(0,2) === '\r');
    };

    return Filter;

})();