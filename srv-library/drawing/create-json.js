(function () {
    function getOffset(el) {
        var _x = 0;
        var _y = 0;
        while (el && !isNaN(el.offsetLeft) && !isNaN(el.offsetTop)) {
            _x += el.offsetLeft - el.scrollLeft;
            _y += el.offsetTop - el.scrollTop;
            el = el.offsetParent;
        }
        return { top: Math.trunc(_x), left: Math.trunc(_y) };
    }

    function getBorderStyle(node, pos) {
        var hexDigits = new Array("0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f");
        function rgb2hex(rgb) {
            function hex(x) {
                return isNaN(x) ? "00" : hexDigits[(x - x % 16) / 16] + hexDigits[x % 16];
            }

            rgb = rgb.match(/^rgb\((\d+),\s*(\d+),\s*(\d+)\)$/);
            return "#" + hex(rgb[1]) + hex(rgb[2]) + hex(rgb[3]);
        }

        return {
            color: rgb2hex(node.style.getPropertyValue('border-' + pos + '-color')),
            thickness: parseInt(node.style.getPropertyValue('border-' + pos + '-width')),
        }
    }

    let data = {};
    let loc = {};

    let wrap = document.getElementsByClassName('seat_table_wrap')[0];
    let wrapOffset = getOffset(wrap);

    data["width"] = wrap.scrollWidth;
    data["height"] = wrap.scrollHeight;

    ///// background
    let background = document.getElementsByClassName('seat_table_wrap_img');
    if (background.length == 1) {
        let p = getOffset(background[0]);
        loc["background"] = {
            "x": p.top - wrapOffset.top,
            "y": p.left - wrapOffset.left,
        };
        data["background"] = background[0].getAttribute("src").match(/^.+\/([^\/]+)$/)[1];
    }

    ///// exit
    let exit = document.getElementsByClassName('etc_type');
    if (exit.length == 1) {
        let off = getOffset(exit[0]);
        loc["exit"] = {
            "x": off.top - wrapOffset.top,
            "y": off.left - wrapOffset.left,
        };
    }

    ///// seat
    let nodes = document.getElementsByClassName('general_seat');
    for (var index = 0; index < nodes.length; index++) {
        let node = nodes[index];

        let off = getOffset(node);
        loc[node.getElementsByClassName('seat_num')[0].innerText] = {
            "x": off.top - wrapOffset.top,
            "y": off.left - wrapOffset.left,
            "border-top": getBorderStyle(node, 'top'),
            "border-left": getBorderStyle(node, 'left'),
            "border-right": getBorderStyle(node, 'right'),
            "border-bottom": getBorderStyle(node, 'bottom'),
        }
    }

    data["location"] = loc;
    return JSON.stringify(data, null, '    ');
})()