export async function request(path,options={}){
 const response=await fetch(path,{credentials:'same-origin',...options,headers:{'Content-Type':'application/json',...options.headers}});const text=await response.text();let data;try{data=JSON.parse(text)}catch{data={error:{message:text||response.statusText}}}if(!response.ok){const error=new Error(data.error?.message||`Response ${response.status}`);error.code=data.error?.code;error.status=response.status;throw error;}return data;
}
