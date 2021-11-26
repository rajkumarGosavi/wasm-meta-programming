(function () {
    const go = new Go();
    let mod, inst;

    WebAssembly.instantiateStreaming(fetch("../app/lib.wasm"), go.importObject).then((result) => {
        mod = result.module;
        inst = result.instance;

        go.run(inst);
        console.log(helloWorld());
    });

})()

async function instantiate() {
    await go.Run(inst);
    inst = await WebAssembly.instantiate(mod, go.importObject)
}

function generate() {
    let ip = document.getElementById("input-text").value;
    let rawData = new TextEncoder("utf-8").encode(ip)
    let res = generateCode(rawData, "hel");

    let op = document.getElementById("output-text");
    op.value = res;
}