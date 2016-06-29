console.log("editor config loaded");

var editor = CodeMirror.fromTextArea(document.getElementById("code"), {
    theme: "monokai",
    matchBrackets: true,
    mode: "text/x-go",
    lineNumbers:"true"
});
var Terminal = CodeMirror.fromTextArea(document.getElementById("Terminal"), {
   theme: "solarized dark",
   indentUnit: 8,
   tabSize: 8,
   indentWithTabs: true,
});


var lastInputFlag=false;
var readPoint; //point in terminal to read STDIN from 
        // -----------------------websockets-----------------//
        var programSocket = new WebSocket("ws://"+window.location.host+"/run");

  // Write message on receive
  programSocket.onmessage = function(event) {
    var inputMsgType;
    var msg;
    try{
        // console.log(typeof JSON.parse(event.data));
        msg=JSON.parse(event.data);
        inputMsgType="json";

    }catch(e){
        console.log("err "+e);
    }
    if (inputMsgType=="json")
{ //Format and shareLink sends JSON msg
    if (msg.Category=="format")
        {editor.setValue(msg.Data)}
    else if(msg.Category=="share")
    {
        var  shareEle=document.getElementById("shareLink");
        shareEle.value=msg.Data;
        shareEle.setAttribute("style","visibility: visible;");
    }
    else if (msg.Category=="getLink") {
        editor.setValue(msg.Data);
    }
}else if(lastInputFlag){ //"Ignoring copy of STDIN"
lastInputFlag=false;
}else{
    Terminal.setValue(Terminal.getValue()+event.data);
    function LastlineCharCount(){
        lastLineChar=Terminal.getLine(Terminal.lastLine());
        readPoint={"line":Terminal.lastLine(),"char":lastLineChar.length};
        return lastLineChar.length;
    }
    LastlineCharCount();
    Terminal.markText({line:0,ch:0},{line:Terminal.lastLine(),ch:LastlineCharCount()},{className:"blockedInput",readOnly: true}); 
    Terminal.setCursor({line: Terminal.lineCount()});
}
};

programSocket.onclose = function(){
     // websocket is closed.
     alert("Server session is closed..."); 
 };

 function share() {
    var msg={Category:"share",Data:editor.getValue()};
    console.log("SENDING---->"+editor.getValue());
    programSocket.send(JSON.stringify(msg));
}

function loadLink() {
    var url =document.getElementById("inputUrl").value;
    console.log(url);
    if(url!==""){
    var msg={Category:"getLink",Data:url};
    console.log("SENDING---->"+JSON.stringify(msg));
    programSocket.send(JSON.stringify(msg));
}

}


function stopProgram() {
 var msg={category:"stop",Data:"nil"}
 console.log("STOPING-->"+JSON.stringify(msg));
 programSocket.send(JSON.stringify(msg)); 
}

function Format() {
 var content=editor.getValue();
 var msg={category:"format",Data:editor.getValue()}
 console.log("SENDING-->"+JSON.stringify(msg));
 programSocket.send(JSON.stringify(msg));
           // Terminal.setValue(""); //wipe data from last run 
       }


       function Run() {
        var content=editor.getValue();
        var msg={category:"code",Data:editor.getValue()}
        console.log("SENDING-->"+JSON.stringify(msg));
        programSocket.send(JSON.stringify(msg));
Terminal.setValue(""); //wipe data from last run 
}
var Terminalsend= {"Enter": function(cm){
    lastInputFlag=true;
    function RealSTDIN(){
        var rawSTDIN=Terminal.getLine(readPoint.line);
        var STDIN=Array.from(rawSTDIN);
        STDIN.splice(0,readPoint.char);
        STDIN=STDIN.join(''); 
        console.log("STDIN :"+STDIN);
        return STDIN;
    }
    var msg={Category:"input",Data:RealSTDIN()}
    Terminal.setValue(Terminal.getValue()+"\n");
    if(msg.Data!=null){
        console.log("sending-->INPUT"+JSON.stringify(msg));
        programSocket.send(JSON.stringify(msg));
    }else{
        console.log("NOT sending-->INPUT"+JSON.stringify(msg));
        console.log("ERR:NO msg.Data");
    }
}
}

Terminal.addKeyMap(Terminalsend);
