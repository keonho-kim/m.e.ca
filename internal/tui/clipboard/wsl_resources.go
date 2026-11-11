// Purpose: Limit the Windows clipboard process and PNG buffer before native image conversion grows memory.
// Pattern: Embedded C# adapter for built-in Windows Job Objects and bounded streams.
// Usage: The fixed PowerShell script compiles these guards in its ephemeral child process.
// Related: internal/tui/clipboard/wsl_script.go, docs/images.md.
package clipboard

// Windows PowerShell 5.1 uses the .NET Framework compiler. Keep C# syntax at
// version 5 and fail explicitly if the host cannot install the process limit.
// https://learn.microsoft.com/windows/win32/procthread/job-objects
const windowsResourceTypes = `using System;
using System.IO;
using System.ComponentModel;
using System.Runtime.InteropServices;
namespace MecaClipboard {
  public sealed class MemoryGuard : IDisposable {
    [StructLayout(LayoutKind.Sequential)]
    struct BasicLimits {
      public long ProcessTime, JobTime;
      public uint Flags;
      public UIntPtr MinWorkingSet, MaxWorkingSet;
      public uint ActiveProcesses;
      public UIntPtr Affinity;
      public uint Priority, Scheduling;
    }
    [StructLayout(LayoutKind.Sequential)]
    struct Counters { public ulong ReadOps, WriteOps, OtherOps, ReadBytes, WriteBytes, OtherBytes; }
    [StructLayout(LayoutKind.Sequential)]
    struct Limits {
      public BasicLimits Basic;
      public Counters IO;
      public UIntPtr ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory;
    }
    [DllImport("kernel32.dll", EntryPoint="CreateJobObjectW", CharSet=CharSet.Unicode, SetLastError=true)]
    static extern IntPtr CreateJobObject(IntPtr attributes, string name);
    [DllImport("kernel32.dll", SetLastError=true)]
    static extern bool SetInformationJobObject(IntPtr job, int type, ref Limits limits, uint size);
    [DllImport("kernel32.dll", SetLastError=true)]
    static extern bool AssignProcessToJobObject(IntPtr job, IntPtr process);
    [DllImport("kernel32.dll")] static extern IntPtr GetCurrentProcess();
    [DllImport("kernel32.dll")] static extern bool CloseHandle(IntPtr handle);
    IntPtr handle;
    public MemoryGuard(long maximum) {
      handle = CreateJobObject(IntPtr.Zero, null);
      if (handle == IntPtr.Zero) throw new Win32Exception(Marshal.GetLastWin32Error());
      try {
        Limits limits = new Limits();
        limits.Basic.Flags = 0x100; // JOB_OBJECT_LIMIT_PROCESS_MEMORY.
        limits.ProcessMemory = new UIntPtr(checked((ulong)maximum));
        if (!SetInformationJobObject(handle, 9, ref limits, (uint)Marshal.SizeOf(typeof(Limits))) ||
            !AssignProcessToJobObject(handle, GetCurrentProcess()))
          throw new Win32Exception(Marshal.GetLastWin32Error());
      } catch { Dispose(); throw; }
    }
    // The job limits this process until it exits. Closing our handle must not
    // kill the process before its framed stdout has reached the Linux reader.
    public void Dispose() {
      if (handle != IntPtr.Zero) { CloseHandle(handle); handle = IntPtr.Zero; }
    }
  }
  public sealed class BoundedStream : MemoryStream {
    readonly int maximum;
    public bool Limited { get; private set; }
    public BoundedStream(int maximum) { this.maximum = maximum; }
    void Check(long value) {
      if (value < 0 || value > maximum) { Limited = true; throw new IOException(); }
    }
    public override int Capacity {
      get { return base.Capacity; }
      set { base.Capacity = Math.Min(value, maximum); }
    }
    public override long Position {
      get { return base.Position; }
      set { Check(value); base.Position = value; }
    }
    public override long Seek(long offset, SeekOrigin origin) {
      long basis = origin == SeekOrigin.Begin ? 0 : origin == SeekOrigin.Current ? Position : Length;
      long next = checked(basis + offset); Check(next);
      return base.Seek(offset, origin);
    }
    public override void SetLength(long value) { Check(value); base.SetLength(value); }
    public override void Write(byte[] buffer, int offset, int count) {
      Check(checked(Position + count)); base.Write(buffer, offset, count);
    }
    public override void WriteByte(byte value) { Check(checked(Position + 1)); base.WriteByte(value); }
  }
}`
