"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Share2Icon, X, Plus } from "lucide-react";
import { notify } from "@components/Notification";

interface SharePeerModalProps {
  peerId: string;
  peerName: string;
  open: boolean;
  onClose: () => void;
}

const PROTOCOLS = ["ALL", "TCP", "UDP", "TCP+UDP", "HTTP", "HTTPS"];

export function SharePeerModal({ peerId, peerName, open, onClose }: SharePeerModalProps) {
  const [name, setName] = useState("");
  const [targetAccountId, setTargetAccountId] = useState("");
  const [protocol, setProtocol] = useState("ALL");
  const [allPorts, setAllPorts] = useState(true);
  const [portInput, setPortInput] = useState("");
  const [ports, setPorts] = useState<number[]>([]);
  const [ttlHours, setTtlHours] = useState("24");
  const [loading, setLoading] = useState(false);
  const [mounted, setMounted] = useState(false);
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => { setMounted(true); return () => setMounted(false); }, []);

  const canSelectPorts = protocol === "TCP" || protocol === "UDP" || protocol === "TCP+UDP";

  if (!open || !mounted) return null;

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === overlayRef.current) onClose();
  };

  const stopAll = (e: React.SyntheticEvent) => {
    e.stopPropagation();
    e.nativeEvent.stopImmediatePropagation();
  };

  const handleProtocolChange = (p: string) => {
    setProtocol(p);
    if (p !== "TCP" && p !== "UDP") {
      setAllPorts(true);
      setPorts([]);
    }
  };

  const addPort = (e: React.KeyboardEvent | React.MouseEvent) => {
    stopAll(e);
    const p = parseInt(portInput);
    if (p > 0 && p <= 65535 && !ports.includes(p)) {
      setPorts(prev => [...prev, p]);
      setPortInput("");
    }
  };

  const removePort = (p: number) => setPorts(ports.filter(x => x !== p));

  const isValid = name && targetAccountId && (!canSelectPorts || allPorts || ports.length > 0);

  const handleSubmit = async (e: React.MouseEvent) => {
    stopAll(e);
    if (!isValid) return;
    setLoading(true);
    try {
      const body: any = {
        name,
        source_peer_id: peerId,
        target_account_id: targetAccountId,
        protocol: protocol === "ALL" ? "tcp" : protocol === "TCP+UDP" ? "tcp" : protocol.toLowerCase(),
        ttl_hours: parseInt(ttlHours),
        port: allPorts || !canSelectPorts ? 0 : ports[0] || 0,
        ports: allPorts || !canSelectPorts ? [] : ports,
      };
      const res = await fetch("/api/cross-network-exposes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error("Failed");
      notify({ title: "Peer shared successfully", description: `Peer ${peerName} is now shared.` });
      onClose();
    } catch {
      notify({ title: "Error sharing peer", description: "Could not create the peer share." });
    } finally {
      setLoading(false);
    }
  };

  const s = {
    input: { width:"100%", boxSizing:"border-box" as const, backgroundColor:"#2d3035", border:"1px solid #3d4045", borderRadius:"8px", padding:"8px 12px", fontSize:"14px", color:"white", outline:"none" },
    label: { color:"#d1d5db", fontSize:"12px", fontWeight:500 as const, display:"block" as const, marginBottom:"6px" },
  };

  return createPortal(
    <div ref={overlayRef} onClick={handleOverlayClick}
      style={{ position:"fixed",top:0,left:0,right:0,bottom:0,zIndex:99999,display:"flex",alignItems:"center",justifyContent:"center",backgroundColor:"rgba(0,0,0,0.75)",backdropFilter:"blur(2px)" }}>
      <div onClick={stopAll} onMouseDown={stopAll} onMouseUp={stopAll} onKeyDown={stopAll}
        style={{ backgroundColor:"#1a1d20",border:"1px solid #2d3035",borderRadius:"12px",padding:"24px",width:"100%",maxWidth:"480px",boxShadow:"0 25px 50px rgba(0,0,0,0.5)",position:"relative",zIndex:100000 }}>

        <div style={{ display:"flex",alignItems:"center",gap:"12px",marginBottom:"24px" }}>
          <div style={{ padding:"8px",backgroundColor:"#2d3035",borderRadius:"8px" }}>
            <Share2Icon size={18} color="#ff6b35" />
          </div>
          <div>
            <h2 style={{ color:"white",fontWeight:600,fontSize:"16px",margin:0 }}>Share Peer</h2>
            <p style={{ color:"#9ca3af",fontSize:"12px",margin:"2px 0 0 0" }}>{peerName}</p>
          </div>
        </div>

        <div style={{ display:"flex",flexDirection:"column",gap:"16px" }}>

          <div>
            <label style={s.label}>Share Name</label>
            <input type="text" value={name} onChange={e=>setName(e.target.value)} onClick={stopAll} placeholder="e.g. ollama-service" style={s.input} />
          </div>

          <div>
            <label style={s.label}>Target Account ID</label>
            <input type="text" value={targetAccountId} onChange={e=>setTargetAccountId(e.target.value)} onClick={stopAll} placeholder="Account ID to share with" style={s.input} />
          </div>

          {/* Protocol pills */}
          <div>
            <label style={s.label}>Protocol</label>
            <div style={{ display:"flex",gap:"6px",flexWrap:"wrap" }}>
              {PROTOCOLS.map(p => (
                <button key={p} onClick={e=>{stopAll(e);handleProtocolChange(p);}}
                  style={{ padding:"6px 14px",fontSize:"13px",borderRadius:"8px",cursor:"pointer",border:"1px solid",
                    borderColor: protocol===p ? "#e8611a" : "#3d4045",
                    backgroundColor: protocol===p ? "rgba(232,97,26,0.15)" : "#2d3035",
                    color: protocol===p ? "#e8611a" : "#9ca3af",
                    fontWeight: protocol===p ? 600 : 400 }}>
                  {p}
                </button>
              ))}
            </div>
          </div>

          {/* Ports — only for TCP/UDP */}
          {canSelectPorts && (
            <div>
              <label style={s.label}>Ports</label>
              <div style={{ display:"flex",gap:"6px",marginBottom:"10px" }}>
                <button onClick={e=>{stopAll(e);setAllPorts(true);}}
                  style={{ flex:1,padding:"7px",fontSize:"13px",borderRadius:"8px",cursor:"pointer",border:"1px solid",
                    borderColor:allPorts?"#e8611a":"#3d4045",backgroundColor:allPorts?"rgba(232,97,26,0.15)":"#2d3035",
                    color:allPorts?"#e8611a":"#9ca3af",fontWeight:allPorts?600:400 }}>
                  ALL
                </button>
                <button onClick={e=>{stopAll(e);setAllPorts(false);}}
                  style={{ flex:1,padding:"7px",fontSize:"13px",borderRadius:"8px",cursor:"pointer",border:"1px solid",
                    borderColor:!allPorts?"#e8611a":"#3d4045",backgroundColor:!allPorts?"rgba(232,97,26,0.15)":"#2d3035",
                    color:!allPorts?"#e8611a":"#9ca3af",fontWeight:!allPorts?600:400 }}>
                  Select ports
                </button>
              </div>

              {!allPorts && (
                <div>
                  <div style={{ display:"flex",gap:"8px",marginBottom:"8px" }}>
                    <input type="number" value={portInput}
                      onChange={e=>{stopAll(e);setPortInput(e.target.value);}}
                      onClick={stopAll}
                      onKeyDown={e=>{stopAll(e);if(e.key==="Enter")addPort(e);}}
                      placeholder="e.g. 11434" min="1" max="65535"
                      style={{...s.input,flex:1,width:"auto"}} />
                    <button onClick={addPort}
                      style={{ padding:"8px 14px",backgroundColor:"#2d3035",border:"1px solid #3d4045",borderRadius:"8px",color:"white",cursor:"pointer",display:"flex",alignItems:"center",gap:"4px",whiteSpace:"nowrap" }}>
                      <Plus size={14}/> Add
                    </button>
                  </div>
                  <div style={{ display:"flex",flexWrap:"wrap",gap:"6px",minHeight:"24px" }}>
                    {ports.map(p=>(
                      <span key={p} style={{ display:"inline-flex",alignItems:"center",gap:"4px",backgroundColor:"rgba(232,97,26,0.15)",border:"1px solid rgba(232,97,26,0.4)",borderRadius:"6px",padding:"3px 8px",fontSize:"13px",color:"#e8611a" }}>
                        {p}
                        <button onClick={e=>{stopAll(e);removePort(p);}} style={{ background:"none",border:"none",cursor:"pointer",color:"#e8611a",padding:0,display:"flex" }}>
                          <X size={12}/>
                        </button>
                      </span>
                    ))}
                    {ports.length===0 && <span style={{ color:"#6b7280",fontSize:"12px" }}>Add ports above</span>}
                  </div>
                </div>
              )}
            </div>
          )}

          <div>
            <label style={s.label}>Expiration</label>
            <select value={ttlHours} onChange={e=>setTtlHours(e.target.value)} onClick={stopAll} style={s.input}>
              <option value="1">1 hour</option>
              <option value="8">8 hours</option>
              <option value="24">24 hours</option>
              <option value="72">3 days</option>
              <option value="168">7 days</option>
              <option value="0">Never</option>
            </select>
          </div>
        </div>

        <div style={{ display:"flex",gap:"12px",marginTop:"24px" }}>
          <button onClick={e=>{stopAll(e);onClose();}} style={{ flex:1,padding:"10px 16px",fontSize:"14px",color:"#d1d5db",backgroundColor:"#2d3035",border:"1px solid #3d4045",borderRadius:"8px",cursor:"pointer" }}>Cancel</button>
          <button onClick={handleSubmit} disabled={loading||!isValid}
            style={{ flex:1,padding:"10px 16px",fontSize:"14px",color:"white",backgroundColor:"#e8611a",border:"none",borderRadius:"8px",cursor:loading?"wait":"pointer",opacity:loading||!isValid?0.5:1 }}>
            {loading?"Sharing...":"Share Peer"}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
