import create from 'zustand'

export interface UseModalState {
    isOpen: boolean;
    setOpen: (value: boolean) => void;
}

const useModal = create<UseModalState>((set) => ({
    isOpen: true,
    setOpen: (value: boolean) => set({isOpen: value}),
  }));

export default useModal;